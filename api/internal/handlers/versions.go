package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"blackbox-api/internal/audit"
	"blackbox-api/internal/models"
	"blackbox-api/internal/repository"
	"blackbox-api/internal/service"
	"blackbox-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type VersionsHandler struct {
	db          *sql.DB
	minio       *storage.MinIOImprovedClient
	versionRepo repository.VersionRepository
	diffRepo    repository.DiffRepository
	audit       *audit.Logger
}

func NewVersionsHandler(db *sql.DB, minio *storage.MinIOImprovedClient, versionRepo repository.VersionRepository, diffRepo repository.DiffRepository, auditLog *audit.Logger) *VersionsHandler {
	return &VersionsHandler{
		db:          db,
		minio:       minio,
		versionRepo: versionRepo,
		diffRepo:    diffRepo,
		audit:       auditLog,
	}
}

func (h *VersionsHandler) GetVersions(c *gin.Context) {
	rows, err := h.db.QueryContext(c.Request.Context(), `
		SELECT cv.id, cv.device_id, cv.version_hash, cv.storage_type, cv.storage_path,
		       cv.parent_version_id, cv.chain_base_id, cv.chain_position,
		       cv.original_size, cv.compressed_size, cv.created_at,
		       d.hostname
		FROM config_versions cv
		JOIN devices d ON cv.device_id = d.id
		ORDER BY cv.id DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query versions"})
		return
	}
	defer rows.Close()

	var versions []models.VersionWithDevice
	for rows.Next() {
		var v models.VersionWithDevice
		var parentID, chainBaseID sql.NullInt32
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
			&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt,
			&v.DeviceHostname); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan version"})
			return
		}
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func (h *VersionsHandler) GetVersionContent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID"})
		return
	}

	version, err := h.versionRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get version"})
		return
	}
	if version == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	content, err := service.GetCachedVersionContent(c.Request.Context(), h.versionRepo, h.minio, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to reconstruct version: %v", err)})
		return
	}

	computedHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if computedHash != version.VersionHash {
		log.Printf("warning: hash mismatch for version %d: expected %s, got %s", id, version.VersionHash, computedHash)
	}

	u, r, ip := audit.FromGin(c)
	h.audit.Log(u, r, "view_config", map[string]any{"version_id": id, "device_id": version.DeviceID}, ip)
	c.Data(http.StatusOK, "text/plain; charset=utf-8", content)
}

func (h *VersionsHandler) GetVersionDiff(c *gin.Context) {
	id1, err := strconv.Atoi(c.Param("id1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID 1"})
		return
	}

	id2, err := strconv.Atoi(c.Param("id2"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID 2"})
		return
	}

	diffIndex, err := h.diffRepo.GetIndex(c.Request.Context(), id1, id2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query diff index"})
		return
	}

	if diffIndex != nil && diffIndex.DiffStoragePath != nil {
		cached, err := h.minio.DownloadDiffCache(c.Request.Context(), *diffIndex.DiffStoragePath)
		if err == nil {
			var diffLines []models.DiffLine
			if json.Unmarshal(cached, &diffLines) == nil {
				c.JSON(http.StatusOK, models.DiffResult{
					LeftVersionID:  id1,
					RightVersionID: id2,
					Lines:          diffLines,
				})
				return
			}
			log.Printf("failed to unmarshal cached diff, recalculating")
		} else {
			log.Printf("failed to download cached diff: %v, recalculating", err)
		}
	}

	version1, err := h.versionRepo.GetByID(c.Request.Context(), id1)
	if err != nil || version1 == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 1 not found"})
		return
	}

	version2, err := h.versionRepo.GetByID(c.Request.Context(), id2)
	if err != nil || version2 == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 2 not found"})
		return
	}

	content1, err := service.GetCachedVersionContent(c.Request.Context(), h.versionRepo, h.minio, version1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 1: %v", err)})
		return
	}

	content2, err := service.GetCachedVersionContent(c.Request.Context(), h.versionRepo, h.minio, version2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 2: %v", err)})
		return
	}

	diffEngine := storage.NewDiffEngine()
	addedLines, removedLines := diffEngine.ParseUnifiedDiffStats(
		diffEngine.CreateUnifiedDiff(content1, content2, fmt.Sprintf("v%d", id1), fmt.Sprintf("v%d", id2)),
	)
	diffLines := service.ComputeDiffLines(string(content1), string(content2))

	go func() {
		data, err := json.Marshal(diffLines)
		if err != nil {
			log.Printf("failed to marshal diff lines: %v", err)
			return
		}
		path, err := h.minio.UploadDiffCache(context.Background(), id1, id2, data)
		if err != nil {
			log.Printf("failed to upload diff cache: %v", err)
			return
		}
		if err := h.diffRepo.SaveIndex(context.Background(), id1, id2, addedLines, removedLines, path); err != nil {
			log.Printf("failed to save diff index: %v", err)
		}
	}()

	u, r, ip := audit.FromGin(c)
	h.audit.Log(u, r, "view_diff", map[string]any{"version_id_1": id1, "version_id_2": id2}, ip)
	c.JSON(http.StatusOK, models.DiffResult{
		LeftVersionID:  id1,
		RightVersionID: id2,
		Lines:          diffLines,
	})
}

func (h *VersionsHandler) GetDiffByDate(c *gin.Context) {
	deviceIDStr := c.Query("deviceId")
	if deviceIDStr == "" {
		deviceIDStr = c.Query("device_id")
	}
	if deviceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DeviceId is required"})
		return
	}
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deviceId"})
		return
	}

	var id1, id2 int
	date1 := c.Query("date1")
	date2 := c.Query("date2")
	aStr := c.Query("a")
	bStr := c.Query("b")

	if aStr != "" && bStr != "" {
		id1, err = strconv.Atoi(aStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version id a"})
			return
		}
		id2, err = strconv.Atoi(bStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version id b"})
			return
		}
	} else if date1 != "" && date2 != "" {
		id1, id2, err = h.versionRepo.ResolveByDate(c.Request.Context(), deviceID, date1, date2)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide either a&b (version ids) or date1&date2 (YYYY-MM-DD)"})
		return
	}

	v1, err := h.versionRepo.GetByID(c.Request.Context(), id1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version a not found"})
		return
	}
	v2, err := h.versionRepo.GetByID(c.Request.Context(), id2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version b not found"})
		return
	}
	if v1.DeviceID != deviceID || v2.DeviceID != deviceID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Versions must belong to the given device"})
		return
	}

	content1, err := service.GetCachedVersionContent(c.Request.Context(), h.versionRepo, h.minio, v1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 1: %v", err)})
		return
	}
	content2, err := service.GetCachedVersionContent(c.Request.Context(), h.versionRepo, h.minio, v2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 2: %v", err)})
		return
	}

	diffLines := service.ComputeDiffLines(string(content1), string(content2))
	u, r, ip := audit.FromGin(c)
	h.audit.Log(u, r, "view_diff_date", map[string]any{"device_id": deviceID, "date1": date1, "date2": date2}, ip)
	c.JSON(http.StatusOK, models.DiffResult{
		LeftVersionID:  id1,
		RightVersionID: id2,
		LeftContent:    "",
		RightContent:   "",
		Lines:          diffLines,
	})
}
