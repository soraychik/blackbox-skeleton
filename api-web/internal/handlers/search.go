package handlers

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"blackbox-api/internal/models"
	"blackbox-api/internal/repository"
	"blackbox-api/internal/service"
	"blackbox-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	minio       *storage.MinIOImprovedClient
	versionRepo repository.VersionRepository
	deviceRepo  repository.DeviceRepository
	db          *sql.DB
}

func NewSearchHandler(minio *storage.MinIOImprovedClient, versionRepo repository.VersionRepository, deviceRepo repository.DeviceRepository, db *sql.DB) *SearchHandler {
	return &SearchHandler{
		minio:       minio,
		versionRepo: versionRepo,
		deviceRepo:  deviceRepo,
		db:          db,
	}
}

// patternHash возвращает короткий ключ для кэша search_index.
func patternHash(pattern string, caseSensitive bool) string {
	key := pattern
	if caseSensitive {
		key += ":cs"
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))[:16]
}

// getSearchCache проверяет кэш search_index. Возвращает -1 если кэша нет.
func (h *SearchHandler) getSearchCache(ctx context.Context, versionID int, hash string) int {
	var count int
	err := h.db.QueryRowContext(ctx,
		`SELECT match_count FROM search_index WHERE version_id=? AND pattern_hash=?`,
		versionID, hash,
	).Scan(&count)
	if err != nil {
		return -1
	}
	return count
}

// saveSearchCache сохраняет результат в search_index асинхронно.
func (h *SearchHandler) saveSearchCache(versionID int, hash string, count int) {
	go func() {
		_, err := h.db.Exec(
			`INSERT IGNORE INTO search_index (version_id, pattern_hash, match_count) VALUES (?,?,?)`,
			versionID, hash, count,
		)
		if err != nil {
			log.Printf("saveSearchCache: %v", err)
		}
	}()
}

func (h *SearchHandler) PostSearchChanges(c *gin.Context) {
	var req models.SearchChangesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if len(req.AddedPatterns) == 0 && len(req.RemovedPatterns) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one of added_patterns or removed_patterns is required"})
		return
	}

	addedRegex, err := service.CompilePatterns(req.AddedPatterns)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid added_patterns: %v", err)})
		return
	}
	removedRegex, err := service.CompilePatterns(req.RemovedPatterns)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid removed_patterns: %v", err)})
		return
	}

	devices, err := h.deviceRepo.GetAllEnabled(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list devices"})
		return
	}

	searchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	type job struct{ d models.DeviceRow }
	type result struct {
		res models.DeviceChangeResult
		ok  bool
	}

	jobs := make(chan job, len(devices))
	resultsCh := make(chan result, len(devices))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				pairs, err := h.versionRepo.GetPairsByDevice(searchCtx, j.d.ID, req.FromDate, req.ToDate)
				if err != nil {
					log.Printf("GetPairsByDevice device %d: %v", j.d.ID, err)
					resultsCh <- result{}
					continue
				}

				var changeList []models.ChangeMatch
				for _, pair := range pairs {
					content1, err := service.GetCachedVersionContent(searchCtx, h.versionRepo, h.minio, pair.Left)
					if err != nil {
						continue
					}
					content2, err := service.GetCachedVersionContent(searchCtx, h.versionRepo, h.minio, pair.Right)
					if err != nil {
						continue
					}

					diffLines := service.ComputeDiffLines(string(content1), string(content2))
					var addedLines, removedLines []string
					for _, ln := range diffLines {
						switch ln.Type {
						case "added":
							addedLines = append(addedLines, ln.Content)
						case "removed":
							removedLines = append(removedLines, ln.Content)
						}
					}
					addedOK := len(addedRegex) == 0 || service.AnyLineMatches(addedLines, addedRegex)
					removedOK := len(removedRegex) == 0 || service.AnyLineMatches(removedLines, removedRegex)
					if addedOK && removedOK {
						changeList = append(changeList, models.ChangeMatch{
							LeftVersionID:  pair.Left.ID,
							RightVersionID: pair.Right.ID,
							LeftDate:       pair.Left.CreatedAt.Format("2006-01-02"),
							RightDate:      pair.Right.CreatedAt.Format("2006-01-02"),
							AddedCount:     len(addedLines),
							RemovedCount:   len(removedLines),
						})
					}
				}

				if len(changeList) > 0 {
					resultsCh <- result{
						ok: true,
						res: models.DeviceChangeResult{
							DeviceID:   j.d.ID,
							Hostname:   j.d.Hostname,
							MgmtIP:     j.d.MgmtIP,
							Vendor:     j.d.Vendor,
							Model:      j.d.Model,
							ChangeList: changeList,
						},
					}
				} else {
					resultsCh <- result{}
				}
			}
		}()
	}

	for _, d := range devices {
		jobs <- job{d: d}
	}
	close(jobs)

	wg.Wait()
	close(resultsCh)

	var results []models.DeviceChangeResult
	for r := range resultsCh {
		if r.ok {
			results = append(results, r.res)
		}
	}

	c.JSON(http.StatusOK, gin.H{"devices": results})
}

func (h *SearchHandler) PostSearchCount(c *gin.Context) {
	var req models.SearchCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.Pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pattern is required"})
		return
	}

	re, err := service.CompileSearchRegex(req.Pattern, req.CaseSensitive)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid pattern: %v", err)})
		return
	}

	devices, err := h.deviceRepo.GetAllEnabledWithLatestVersion(c.Request.Context(), req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query devices"})
		return
	}

	searchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pHash := patternHash(req.Pattern, req.CaseSensitive)

	type job struct{ dv models.DeviceVersionRow }
	type result struct {
		res models.SearchCountResult
		ok  bool
	}

	jobs := make(chan job, len(devices))
	resultsCh := make(chan result, len(devices))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				// Оптимизация 1: проверяем кэш search_index до скачивания контента.
				// Если паттерн уже считался для этой версии — возвращаем из БД мгновенно.
				if cached := h.getSearchCache(searchCtx, j.dv.VersionID, pHash); cached >= 0 {
					if cached == 0 {
						resultsCh <- result{}
					} else {
						resultsCh <- result{
							ok: true,
							res: models.SearchCountResult{
								DeviceID:   j.dv.DeviceID,
								VersionID:  j.dv.VersionID,
								Hostname:   j.dv.Hostname,
								MgmtIP:     j.dv.MgmtIP,
								MatchCount: cached,
								// Сниппеты не кэшируем — их мало и они быстро считаются
							},
						}
					}
					continue
				}

				// Оптимизация 2: собираем ConfigVersion прямо из DeviceVersionRow
				// без лишнего GetByID запроса к БД.
				version := &models.ConfigVersion{
					ID:          j.dv.VersionID,
					DeviceID:    j.dv.DeviceID,
					StorageType: j.dv.StorageType,
					StoragePath: j.dv.StoragePath,
				}

				content, err := service.GetCachedVersionContent(searchCtx, h.versionRepo, h.minio, version)
				if err != nil {
					log.Printf("failed to get content for device %d: %v", j.dv.DeviceID, err)
					resultsCh <- result{}
					continue
				}

				contentStr := string(content)
				matches := re.FindAllStringIndex(contentStr, -1)

				// Сохраняем результат в кэш (даже если 0 совпадений)
				h.saveSearchCache(j.dv.VersionID, pHash, len(matches))

				if len(matches) == 0 {
					resultsCh <- result{}
					continue
				}

				// Оптимизация 3: строим индекс смещений один раз,
				// вместо двойного прохода по строкам для каждого match.
				lines := strings.Split(contentStr, "\n")
				snippetLines := service.FindSnippetLinesOptimized(lines, matches, 2)

				resultsCh <- result{
					ok: true,
					res: models.SearchCountResult{
						DeviceID:   j.dv.DeviceID,
						VersionID:  j.dv.VersionID,
						Hostname:   j.dv.Hostname,
						MgmtIP:     j.dv.MgmtIP,
						MatchCount: len(matches),
						Snippets:   snippetLines,
					},
				}
			}
		}()
	}

	for _, dv := range devices {
		jobs <- job{dv: dv}
	}
	close(jobs)

	wg.Wait()
	close(resultsCh)

	var results []models.SearchCountResult
	for r := range resultsCh {
		if r.ok {
			results = append(results, r.res)
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}