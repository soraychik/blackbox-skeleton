package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"blackbox-api/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var dbPool *sql.DB

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("failed to initialize database pool: %v", err)
	}
	defer dbPool.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// Сжатие ответов (особенно контент конфигов 800KB+) — быстрее передача по сети
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "BlackBox API Web is running...")
	})

	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	router.GET("/devices", getDevices)
	router.GET("/devices/:id", getDeviceByID)
	router.GET("/devices/:id/versions", getDeviceVersions)
	router.GET("/devices/compare/latest", getLatestVersionsForDevices)
	router.GET("/dashboard/stats", getDashboardStats)
	router.GET("/versions", getVersions)
	router.GET("/versions/:id/content", getVersionContent)
	router.GET("/versions/diff/:id1/:id2", getVersionDiff)
	// ТЗ 2.3: UC-2 — сравнение конфигурации устройства между датами
	router.GET("/diff/date", getDiffByDate)
	// ТЗ 2.3: UC-4 — выгрузка конфига за выбранную дату
	router.GET("/export/config", getExportConfig)
	// ТЗ 2.1 UC-1 — поиск устройств по изменениям (добавились/удалились строки по шаблонам)
	router.POST("/search/changes", postSearchChanges)
	// ТЗ 2.3 UC-5 — поиск по конфигурациям (regexp, сниппеты)
	router.POST("/search/count", postSearchCount)
	// Принудительный запуск сканирования (прокси к scheduler)
	router.POST("/scan", postTriggerScan)
	router.GET("/scan/status", getScanStatus)

	log.Println("api web server starting on :8080")
	router.Run(":8080")
}

func initDB() error {
	dbHost := getEnv("DATABASE_HOST", "mysql-db")
	dbPort := getEnv("DATABASE_PORT", "3306")
	dbUser := getEnv("DATABASE_USER", "appuser")
	dbPassword := getEnv("DATABASE_PASSWORD", "apppassword")
	dbName := getEnv("DATABASE_NAME", "blackbox")

	dsn := dbUser + ":" + dbPassword + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?parseTime=true"
	log.Printf("connecting to mysql: %s", dsn)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(10)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return err
	}

	log.Println("successfully connected to mysql database")
	dbPool = conn
	return nil
}

func NewDB() (*sql.DB, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}
	return dbPool, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// postTriggerScan вызывает принудительное сканирование в scheduler и сбрасывает таймер следующего автоматического сканирования.
func postTriggerScan(c *gin.Context) {
	schedulerURL := getEnv("SCHEDULER_TRIGGER_URL", "http://scheduler:9090")
	url := strings.TrimSuffix(schedulerURL, "/") + "/scan"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("trigger scan request to scheduler failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Scheduler unreachable"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}
	c.Data(resp.StatusCode, "application/json", body)
}

// getScanStatus возвращает статус текущего сканирования из scheduler.
func getScanStatus(c *gin.Context) {
	schedulerURL := getEnv("SCHEDULER_TRIGGER_URL", "http://scheduler:9090")
	url := strings.TrimSuffix(schedulerURL, "/") + "/status"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("scan status request to scheduler failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "scheduler unreachable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

type Device struct {
	ID        int       `json:"id"`
	Hostname  string    `json:"hostname"`
	MgmtIP    *string   `json:"mgmt_ip,omitempty"`
	Vendor    *string   `json:"vendor,omitempty"`
	Model     *string   `json:"model,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type ConfigVersion struct {
	ID              int       `json:"id"`
	DeviceID        int       `json:"device_id"`
	VersionHash     string    `json:"version_hash"`
	StorageType     string    `json:"storage_type"`
	StoragePath     string    `json:"storage_path"`
	ParentVersionID *int      `json:"parent_version_id,omitempty"`
	ChainBaseID     *int      `json:"chain_base_id,omitempty"`
	ChainPosition   int       `json:"chain_position"`
	OriginalSize    uint32    `json:"original_size"`
	CompressedSize  uint32    `json:"compressed_size"`
	CreatedAt       time.Time `json:"created_at"`
}

type DiffIndex struct {
	ID             int       `json:"id"`
	LeftVersionID  int       `json:"left_version_id"`
	RightVersionID int       `json:"right_version_id"`
	AddedLines     int       `json:"added_lines"`
	RemovedLines   int       `json:"removed_lines"`
	DiffContent    *string   `json:"diff_content,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// versionContentCache — кэш собранного контента версий (просмотр/сравнение открываются быстрее при повторе).
const versionContentCacheMaxEntries = 50

type versionContentCache struct {
	mu    sync.RWMutex
	byID  map[int][]byte
	order []int
}

func newVersionContentCache() *versionContentCache {
	return &versionContentCache{byID: make(map[int][]byte)}
}

func (c *versionContentCache) Get(id int) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	// Возвращаем копию, чтобы кэш не меняли снаружи
	out := make([]byte, len(b))
	copy(out, b)
	return out, true
}

func (c *versionContentCache) Set(id int, content []byte) {
	if len(content) == 0 {
		return
	}
	clone := make([]byte, len(content))
	copy(clone, content)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.byID[id]; exists {
		c.byID[id] = clone
		return
	}
	for len(c.order) >= versionContentCacheMaxEntries {
		evict := c.order[0]
		c.order = c.order[1:]
		delete(c.byID, evict)
	}
	c.byID[id] = clone
	c.order = append(c.order, id)
}

var globalContentCache = newVersionContentCache()

func getDevices(c *gin.Context) {
	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	rows, err := db.Query("SELECT id, hostname, mgmt_ip, vendor, model, enabled, created_at FROM devices ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query devices"})
		return
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.MgmtIP, &d.Vendor, &d.Model, &d.Enabled, &d.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan device"})
			return
		}
		devices = append(devices, d)
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func getDeviceByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	var d Device
	err = db.QueryRow(
		"SELECT id, hostname, mgmt_ip, vendor, model, enabled, created_at FROM devices WHERE id = ?",
		id,
	).Scan(&d.ID, &d.Hostname, &d.MgmtIP, &d.Vendor, &d.Model, &d.Enabled, &d.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": d})
}

func getVersions(c *gin.Context) {
	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	rows, err := db.Query(`
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

	type VersionWithDevice struct {
		ID             int       `json:"id"`
		DeviceID       int       `json:"device_id"`
		DeviceHostname string    `json:"device_hostname"`
		VersionHash    string    `json:"version_hash"`
		StorageType    string    `json:"storage_type"`
		StoragePath    string    `json:"storage_path"`
		ChainPosition  int       `json:"chain_position"`
		OriginalSize   uint32    `json:"original_size"`
		CompressedSize uint32    `json:"compressed_size"`
		CreatedAt      time.Time `json:"created_at"`
	}

	var versions []VersionWithDevice
	for rows.Next() {
		var v VersionWithDevice
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

// getDashboardStats возвращает агрегированную статистику для дашборда (без LIMIT по версиям).
func getDashboardStats(c *gin.Context) {
	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	var totalDevices int
	if err := db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalDevices); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count devices"})
		return
	}

	var updatedToday int
	if err := db.QueryRow("SELECT COUNT(*) FROM config_versions WHERE DATE(created_at) = CURDATE()").Scan(&updatedToday); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count versions updated today"})
		return
	}

	var devicesWithChanges int
	if err := db.QueryRow("SELECT COUNT(DISTINCT device_id) FROM config_versions").Scan(&devicesWithChanges); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count devices with changes"})
		return
	}

	rows, err := db.Query(`
		SELECT d.id, d.hostname, COUNT(cv.id) AS change_count, MAX(cv.created_at) AS last_change
		FROM devices d
		JOIN config_versions cv ON cv.device_id = d.id
		GROUP BY d.id, d.hostname
		ORDER BY change_count DESC
		LIMIT 5
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query top devices"})
		return
	}
	defer rows.Close()

	type TopDevice struct {
		DeviceID    int       `json:"device_id"`
		Hostname    string    `json:"hostname"`
		ChangeCount int       `json:"change_count"`
		LastChange  time.Time `json:"last_change"`
	}
	var topDevices []TopDevice
	for rows.Next() {
		var t TopDevice
		if err := rows.Scan(&t.DeviceID, &t.Hostname, &t.ChangeCount, &t.LastChange); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan top device"})
			return
		}
		topDevices = append(topDevices, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_devices":        totalDevices,
		"updated_today":        updatedToday,
		"devices_with_changes": devicesWithChanges,
		"top_devices":          topDevices,
	})
}

func getDeviceVersions(c *gin.Context) {
	deviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	// Загружаем устройство для ответа (по ТЗ 2.3)
	var d Device
	err = db.QueryRow(
		"SELECT id, hostname, mgmt_ip, vendor, model, enabled, created_at FROM devices WHERE id = ?",
		deviceID,
	).Scan(&d.ID, &d.Hostname, &d.MgmtIP, &d.Vendor, &d.Model, &d.Enabled, &d.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")

	query := `SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE device_id = ?`

	args := []interface{}{deviceID}

	// Поддержка from/to как даты YYYY-MM-DD (ТЗ 2.3: GET /devices/{id}/versions?from=...&to=...)
	if fromStr != "" && len(fromStr) == 10 && fromStr[4] == '-' && fromStr[7] == '-' {
		query += " AND DATE(created_at) >= ?"
		args = append(args, fromStr)
	} else if fromStr != "" {
		if fromID, parseErr := strconv.Atoi(fromStr); parseErr == nil {
			query += " AND id >= ?"
			args = append(args, fromID)
		}
	}
	if toStr != "" && len(toStr) == 10 && toStr[4] == '-' && toStr[7] == '-' {
		query += " AND DATE(created_at) <= ?"
		args = append(args, toStr)
	} else if toStr != "" {
		if toID, parseErr := strconv.Atoi(toStr); parseErr == nil {
			query += " AND id <= ?"
			args = append(args, toID)
		}
	}

	query += " ORDER BY id DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query versions"})
		return
	}
	defer rows.Close()

	var versions []ConfigVersion
	for rows.Next() {
		var v ConfigVersion
		var parentID, chainBaseID sql.NullInt32
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
			&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan version"})
			return
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			v.ParentVersionID = &pid
		}
		if chainBaseID.Valid {
			cid := int(chainBaseID.Int32)
			v.ChainBaseID = &cid
		}
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, gin.H{"device": d, "versions": versions})
}

func getLatestVersionsForDevices(c *gin.Context) {
	leftID, err := strconv.Atoi(c.Query("leftDeviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid left device ID"})
		return
	}
	rightID, err := strconv.Atoi(c.Query("rightDeviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid right device ID"})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	loadDevice := func(deviceID int) (*Device, error) {
		var d Device
		err := db.QueryRow(
			"SELECT id, hostname, mgmt_ip, vendor, model, enabled, created_at FROM devices WHERE id = ?",
			deviceID,
		).Scan(&d.ID, &d.Hostname, &d.MgmtIP, &d.Vendor, &d.Model, &d.Enabled, &d.CreatedAt)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return &d, nil
	}

	loadLatestVersion := func(deviceID int) (*ConfigVersion, error) {
		var v ConfigVersion
		var parentID, chainBaseID sql.NullInt32
		err := db.QueryRow(`
			SELECT id, device_id, version_hash, storage_type, storage_path,
			       parent_version_id, chain_base_id, chain_position,
			       original_size, compressed_size, created_at
			FROM config_versions
			WHERE device_id = ?
			ORDER BY id DESC
			LIMIT 1`,
			deviceID,
		).Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
			&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			v.ParentVersionID = &pid
		}
		if chainBaseID.Valid {
			cid := int(chainBaseID.Int32)
			v.ChainBaseID = &cid
		}
		return &v, nil
	}

	leftDevice, err := loadDevice(leftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get left device"})
		return
	}
	if leftDevice == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Left device not found"})
		return
	}
	rightDevice, err := loadDevice(rightID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get right device"})
		return
	}
	if rightDevice == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Right device not found"})
		return
	}

	leftVersion, err := loadLatestVersion(leftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get left latest version"})
		return
	}
	rightVersion, err := loadLatestVersion(rightID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get right latest version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"left": gin.H{
			"device":  leftDevice,
			"version": leftVersion,
		},
		"right": gin.H{
			"device":  rightDevice,
			"version": rightVersion,
		},
	})
}

func getVersionContent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID"})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	var version ConfigVersion
	var parentID, chainBaseID sql.NullInt32
	err = db.QueryRow(`
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions WHERE id = ?`, id).Scan(
		&version.ID, &version.DeviceID, &version.VersionHash, &version.StorageType, &version.StoragePath,
		&parentID, &chainBaseID, &version.ChainPosition, &version.OriginalSize, &version.CompressedSize, &version.CreatedAt,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get version"})
		return
	}

	if parentID.Valid {
		pid := int(parentID.Int32)
		version.ParentVersionID = &pid
	}
	if chainBaseID.Valid {
		cid := int(chainBaseID.Int32)
		version.ChainBaseID = &cid
	}

	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	content, err := getCachedVersionContent(c.Request.Context(), db, minioClient, &version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to reconstruct version: %v", err)})
		return
	}

	computedHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if computedHash != version.VersionHash {
		log.Printf("warning: hash mismatch for version %d: expected %s, got %s", id, version.VersionHash, computedHash)
	}

	c.Data(http.StatusOK, "text/plain; charset=utf-8", content)
}

// getCachedVersionContent возвращает контент версии (из кэша или после сборки), заполняет кэш при промахе.
func getCachedVersionContent(ctx context.Context, db *sql.DB, minioClient *storage.MinIOImprovedClient, version *ConfigVersion) ([]byte, error) {
	if content, ok := globalContentCache.Get(version.ID); ok {
		return content, nil
	}
	content, err := reconstructVersionContent(ctx, db, minioClient, version)
	if err != nil {
		return nil, err
	}
	globalContentCache.Set(version.ID, content)
	return content, nil
}

func reconstructVersionContent(ctx context.Context, db *sql.DB, minioClient *storage.MinIOImprovedClient, version *ConfigVersion) ([]byte, error) {
	log.Printf("debug: reconstructVersionContent called with StorageType=%q, ParentVersionID=%v, StoragePath=%q",
		version.StorageType, version.ParentVersionID, version.StoragePath)

	if version.StorageType == "base" {
		return minioClient.DownloadConfig(ctx, version.StoragePath)
	}

	if version.StorageType == "diff" && version.ParentVersionID != nil {
		parentVersion, err := getVersionByID(db, *version.ParentVersionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent version: %w", err)
		}

		baseContent, err := getCachedVersionContent(ctx, db, minioClient, parentVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct parent: %w", err)
		}

		patchData, err := minioClient.DownloadConfig(ctx, version.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to download patch: %w", err)
		}

		diffEngine := storage.NewDiffEngine()
		patchContent := string(patchData)

		return diffEngine.ApplyDiff(baseContent, patchContent)
	}

	return nil, fmt.Errorf("unknown storage type: %s", version.StorageType)
}

func getVersionByID(db *sql.DB, id int) (*ConfigVersion, error) {
	var v ConfigVersion
	var parentID, chainBaseID sql.NullInt32
	err := db.QueryRow(`
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions WHERE id = ?`, id).Scan(
		&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
		&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		pid := int(parentID.Int32)
		v.ParentVersionID = &pid
	}
	if chainBaseID.Valid {
		cid := int(chainBaseID.Int32)
		v.ChainBaseID = &cid
	}
	return &v, nil
}

type DiffLine struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	LineNum  int    `json:"line_num"`
	LeftNum  int    `json:"left_num,omitempty"`
	RightNum int    `json:"right_num,omitempty"`
}

type DiffResult struct {
	LeftVersionID  int        `json:"left_version_id"`
	RightVersionID int        `json:"right_version_id"`
	LeftContent    string     `json:"left_content"`
	RightContent   string     `json:"right_content"`
	Lines          []DiffLine `json:"lines"`
}

func getVersionDiff(c *gin.Context) {
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

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	var diffIndex DiffIndex
	err = db.QueryRow(`
		SELECT id, left_version_id, right_version_id, added_lines, removed_lines, diff_content, created_at 
		FROM diff_index 
		WHERE left_version_id = ? AND right_version_id = ?`,
		id1, id2,
	).Scan(&diffIndex.ID, &diffIndex.LeftVersionID, &diffIndex.RightVersionID,
		&diffIndex.AddedLines, &diffIndex.RemovedLines, &diffIndex.DiffContent, &diffIndex.CreatedAt)

	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query diff index"})
		return
	}

	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	version1, err := getVersionByID(db, id1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 1 not found"})
		return
	}

	version2, err := getVersionByID(db, id2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 2 not found"})
		return
	}

	content1, err := getCachedVersionContent(c.Request.Context(), db, minioClient, version1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 1: %v", err)})
		return
	}

	content2, err := getCachedVersionContent(c.Request.Context(), db, minioClient, version2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 2: %v", err)})
		return
	}

	diffEngine := storage.NewDiffEngine()
	addedLines, removedLines := diffEngine.ParseUnifiedDiffStats(string(content2))

	if diffIndex.ID == 0 {
		diffStr := diffEngine.CreateUnifiedDiff(content1, content2, fmt.Sprintf("v%d", id1), fmt.Sprintf("v%d", id2))
		_, err = db.Exec(`
			INSERT INTO diff_index (left_version_id, right_version_id, added_lines, removed_lines, diff_content) 
			VALUES (?, ?, ?, ?, ?)`,
			id1, id2, addedLines, removedLines, diffStr,
		)
		if err != nil {
			log.Printf("failed to cache diff: %v", err)
		}
	}

	diffLines := computeDiffLines(string(content1), string(content2))

	result := DiffResult{
		LeftVersionID:  id1,
		RightVersionID: id2,
		LeftContent:    "",
		RightContent:   "",
		Lines:          diffLines,
	}

	c.JSON(http.StatusOK, result)
}

const CONTEXT_LINES = 3

func computeDiffLines(text1, text2 string) []DiffLine {
	dmp := diffmatchpatch.New()
	lineChar1, lineChar2, lineArray := dmp.DiffLinesToChars(text1, text2)
	diffs := dmp.DiffMain(lineChar1, lineChar2, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	type tempLine struct {
		Type      string
		Content   string
		LeftNum   int
		RightNum  int
		isChanged bool
	}

	var allLines []tempLine
	leftLineNum := 1
	rightLineNum := 1

	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			tl := tempLine{LeftNum: leftLineNum, RightNum: rightLineNum}
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				tl.Type = "unchanged"
				tl.Content = line
				leftLineNum++
				rightLineNum++
			case diffmatchpatch.DiffDelete:
				tl.Type = "removed"
				tl.Content = line
				tl.isChanged = true
				leftLineNum++
			case diffmatchpatch.DiffInsert:
				tl.Type = "added"
				tl.Content = line
				tl.isChanged = true
				rightLineNum++
			}
			allLines = append(allLines, tl)
		}
	}

	hasChanges := false
	for _, line := range allLines {
		if line.isChanged {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		var result []DiffLine
		for _, tl := range allLines {
			result = append(result, DiffLine{
				Type:     tl.Type,
				Content:  tl.Content,
				LeftNum:  tl.LeftNum,
				RightNum: tl.RightNum,
			})
		}
		return result
	}

	seenIdx := make(map[int]bool)
	for i, line := range allLines {
		if !line.isChanged {
			continue
		}
		start := i - CONTEXT_LINES
		if start < 0 {
			start = 0
		}
		end := i + CONTEXT_LINES + 1
		if end > len(allLines) {
			end = len(allLines)
		}
		for j := start; j < end; j++ {
			seenIdx[j] = true
		}
	}

	var result []DiffLine
	for i, tl := range allLines {
		if seenIdx[i] {
			result = append(result, DiffLine{
				Type:     tl.Type,
				Content:  tl.Content,
				LeftNum:  tl.LeftNum,
				RightNum: tl.RightNum,
			})
		}
	}

	return result
}

// getDiffByDate — UC-2: сравнение конфигурации устройства между датами (ТЗ 2.3: GET /diff/date)
// Поддерживает: ?deviceId=...&a=verId1&b=verId2  или  ?deviceId=...&date1=YYYY-MM-DD&date2=YYYY-MM-DD
func getDiffByDate(c *gin.Context) {
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
		id1, id2, err = resolveDeviceVersionsByDate(c.Request.Context(), deviceID, date1, date2)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide either a&b (version ids) or date1&date2 (YYYY-MM-DD)"})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	v1, err := getVersionByID(db, id1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version a not found"})
		return
	}
	v2, err := getVersionByID(db, id2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version b not found"})
		return
	}
	if v1.DeviceID != deviceID || v2.DeviceID != deviceID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Versions must belong to the given device"})
		return
	}

	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	content1, err := getCachedVersionContent(c.Request.Context(), db, minioClient, v1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 1: %v", err)})
		return
	}
	content2, err := getCachedVersionContent(c.Request.Context(), db, minioClient, v2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 2: %v", err)})
		return
	}

	diffLines := computeDiffLines(string(content1), string(content2))
	result := DiffResult{
		LeftVersionID:  id1,
		RightVersionID: id2,
		LeftContent:    "",
		RightContent:   "",
		Lines:          diffLines,
	}
	c.JSON(http.StatusOK, result)
}

func resolveDeviceVersionsByDate(ctx context.Context, deviceID int, date1, date2 string) (verID1, verID2 int, err error) {
	db, err := NewDB()
	if err != nil {
		return 0, 0, fmt.Errorf("database connection failed")
	}

	// Версия «на дату» — последняя версия с created_at в эту дату (по ТЗ — «конфиг за выбранную дату»)
	for _, d := range []string{date1, date2} {
		if len(d) != 10 || d[4] != '-' || d[7] != '-' {
			return 0, 0, fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}
	}

	var v1ID, v2ID sql.NullInt64
	err = db.QueryRowContext(ctx, `
		SELECT (SELECT id FROM config_versions WHERE device_id = ? AND DATE(created_at) = ? ORDER BY created_at DESC LIMIT 1) AS v1,
		       (SELECT id FROM config_versions WHERE device_id = ? AND DATE(created_at) = ? ORDER BY created_at DESC LIMIT 1) AS v2`,
		deviceID, date1, deviceID, date2,
	).Scan(&v1ID, &v2ID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve versions by date: %w", err)
	}
	lastDate, _ := getLastVersionDate(db, deviceID)
	if !v1ID.Valid {
		if lastDate != "" {
			return 0, 0, fmt.Errorf("no version for date %s; last config registered: %s", date1, lastDate)
		}
		return 0, 0, fmt.Errorf("no version for date %s", date1)
	}
	if !v2ID.Valid {
		if lastDate != "" {
			return 0, 0, fmt.Errorf("no version for date %s; last config registered: %s", date2, lastDate)
		}
		return 0, 0, fmt.Errorf("no version for date %s", date2)
	}
	return int(v1ID.Int64), int(v2ID.Int64), nil
}

// getLastVersionDate возвращает дату (YYYY-MM-DD) последней зарегистрированной версии конфига устройства.
func getLastVersionDate(db *sql.DB, deviceID int) (string, error) {
	var lastDate sql.NullString
	err := db.QueryRow(`
		SELECT DATE_FORMAT(MAX(created_at), '%Y-%m-%d') FROM config_versions WHERE device_id = ?
	`, deviceID).Scan(&lastDate)
	if err != nil || !lastDate.Valid {
		return "", err
	}
	return lastDate.String, nil
}

// getExportConfig — UC-4: выгрузка конфига за выбранную дату (ТЗ 2.3: GET /export/config)
func getExportConfig(c *gin.Context) {
	deviceIDStr := c.Query("deviceId")
	if deviceIDStr == "" {
		deviceIDStr = c.Query("device_id")
	}
	dateStr := c.Query("date")
	if deviceIDStr == "" || dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DeviceId and date (YYYY-MM-DD) are required"})
		return
	}
	if len(dateStr) != 10 || dateStr[4] != '-' || dateStr[7] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date must be YYYY-MM-DD"})
		return
	}
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deviceId"})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	var versionID int
	var hostname string
	err = db.QueryRowContext(c.Request.Context(), `
		SELECT cv.id, d.hostname FROM config_versions cv
		JOIN devices d ON d.id = cv.device_id
		WHERE cv.device_id = ? AND DATE(cv.created_at) = ?
		ORDER BY cv.created_at DESC LIMIT 1`,
		deviceID, dateStr,
	).Scan(&versionID, &hostname)
	if err == sql.ErrNoRows {
		lastDate, _ := getLastVersionDate(db, deviceID)
		payload := gin.H{"error": "No config for this device on the given date"}
		if lastDate != "" {
			payload["last_registered_date"] = lastDate
		}
		c.JSON(http.StatusNotFound, payload)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get version"})
		return
	}

	version, err := getVersionByID(db, versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}
	content, err := getCachedVersionContent(c.Request.Context(), db, minioClient, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get config content: %v", err)})
		return
	}

	filename := fmt.Sprintf("config_%s_%s.txt", strings.ReplaceAll(hostname, " ", "_"), dateStr)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "text/plain", content)
}

// SearchChangesRequest — тело POST /search/changes (UC-1)
type SearchChangesRequest struct {
	AddedPatterns   []string `json:"added_patterns"`
	RemovedPatterns []string `json:"removed_patterns"`
	FromDate        string   `json:"from_date"`
	ToDate          string   `json:"to_date"`
}

// SearchCountRequest — тело POST /search/count (UC-5)
type SearchCountRequest struct {
	Pattern       string `json:"pattern"`
	CaseSensitive bool   `json:"caseSensitive"`
	Scope         string `json:"scope"` // "all" или "device"
	DeviceID      *int   `json:"deviceId"`
}

// SearchCountResult — результат поиска для одного устройства
type SearchCountResult struct {
	DeviceID   int           `json:"device_id"`
	VersionID  int           `json:"version_id"`
	Hostname   string        `json:"hostname"`
	MgmtIP     *string       `json:"mgmt_ip,omitempty"`
	MatchCount int           `json:"match_count"`
	Snippets   []SnippetLine `json:"snippets"`
}

// SnippetLine — одна строка сниппета
type SnippetLine struct {
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

// ChangeMatch — одно совпадение изменений по устройству
type ChangeMatch struct {
	LeftVersionID  int    `json:"left_version_id"`
	RightVersionID int    `json:"right_version_id"`
	LeftDate       string `json:"left_date"`
	RightDate      string `json:"right_date"`
	AddedCount     int    `json:"added_count"`
	RemovedCount   int    `json:"removed_count"`
}

// DeviceChangeResult — устройство и список подходящих изменений
type DeviceChangeResult struct {
	DeviceID   int           `json:"device_id"`
	Hostname   string        `json:"hostname"`
	MgmtIP     *string       `json:"mgmt_ip,omitempty"`
	Vendor     *string       `json:"vendor,omitempty"`
	Model      *string       `json:"model,omitempty"`
	ChangeList []ChangeMatch `json:"changes"`
}

// postSearchChanges — UC-1: найти устройства, у которых конфиг изменился по шаблонам (добавились/удалились строки)
func postSearchChanges(c *gin.Context) {
	var req SearchChangesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if len(req.AddedPatterns) == 0 && len(req.RemovedPatterns) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one of added_patterns or removed_patterns is required"})
		return
	}

	addedRegex, err := compilePatterns(req.AddedPatterns)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid added_patterns: %v", err)})
		return
	}
	removedRegex, err := compilePatterns(req.RemovedPatterns)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid removed_patterns: %v", err)})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	deviceRows, err := db.Query("SELECT id, hostname, mgmt_ip, vendor, model FROM devices WHERE enabled = 1 ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list devices"})
		return
	}
	defer deviceRows.Close()

	var results []DeviceChangeResult
	for deviceRows.Next() {
		var deviceID int
		var hostname string
		var mgmtIP, vendor, model sql.NullString
		if err := deviceRows.Scan(&deviceID, &hostname, &mgmtIP, &vendor, &model); err != nil {
			continue
		}

		versions, err := getDeviceVersionPairs(db, deviceID, req.FromDate, req.ToDate)
		if err != nil {
			log.Printf("getDeviceVersionPairs device %d: %v", deviceID, err)
			continue
		}

		var changeList []ChangeMatch
		for _, pair := range versions {
			content1, err := getCachedVersionContent(c.Request.Context(), db, minioClient, pair.Left)
			if err != nil {
				continue
			}
			content2, err := getCachedVersionContent(c.Request.Context(), db, minioClient, pair.Right)
			if err != nil {
				continue
			}
			diffLines := computeDiffLines(string(content1), string(content2))
			var addedLines, removedLines []string
			for _, ln := range diffLines {
				if ln.Type == "added" {
					addedLines = append(addedLines, ln.Content)
				} else if ln.Type == "removed" {
					removedLines = append(removedLines, ln.Content)
				}
			}
			addedOK := len(addedRegex) == 0 || anyLineMatches(addedLines, addedRegex)
			removedOK := len(removedRegex) == 0 || anyLineMatches(removedLines, removedRegex)
			if addedOK && removedOK {
				changeList = append(changeList, ChangeMatch{
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
			var ip, vdr, mdl *string
			if mgmtIP.Valid {
				ip = &mgmtIP.String
			}
			if vendor.Valid {
				vdr = &vendor.String
			}
			if model.Valid {
				mdl = &model.String
			}
			results = append(results, DeviceChangeResult{
				DeviceID:   deviceID,
				Hostname:   hostname,
				MgmtIP:     ip,
				Vendor:     vdr,
				Model:      mdl,
				ChangeList: changeList,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"devices": results})
}

// postSearchCount — UC-5: поиск по конфигурациям устройств (ТЗ 2.3)
func postSearchCount(c *gin.Context) {
	var req SearchCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.Pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pattern is required"})
		return
	}

	flags := ""
	if !req.CaseSensitive {
		flags = "(?i)"
	}
	pattern := flags + req.Pattern

	re, err := regexp.Compile(pattern)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid pattern: %v", err)})
		return
	}

	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	var query string
	var args []interface{}

	if req.Scope == "device" && req.DeviceID != nil {
		query = `
			SELECT d.id, d.hostname, d.mgmt_ip, cv.id, cv.storage_type, cv.storage_path
			FROM devices d
			JOIN config_versions cv ON cv.id = (
				SELECT id FROM config_versions
				WHERE device_id = d.id
				ORDER BY created_at DESC
				LIMIT 1
			)
			WHERE d.id = ? AND d.enabled = 1`
		args = []interface{}{*req.DeviceID}
	} else {
		query = `
			SELECT d.id, d.hostname, d.mgmt_ip, cv.id, cv.storage_type, cv.storage_path
			FROM devices d
			JOIN config_versions cv ON cv.id = (
				SELECT id FROM config_versions
				WHERE device_id = d.id
				ORDER BY created_at DESC
				LIMIT 1
			)
			WHERE d.enabled = 1
			ORDER BY d.id`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query devices"})
		return
	}
	defer rows.Close()

	type deviceVersion struct {
		deviceID    int
		hostname    string
		mgmtIP      *string
		versionID   int
		storageType string
		storagePath string
	}

	var devices []deviceVersion
	for rows.Next() {
		var dv deviceVersion
		var mgmtIP sql.NullString
		if err := rows.Scan(&dv.deviceID, &dv.hostname, &mgmtIP, &dv.versionID, &dv.storageType, &dv.storagePath); err != nil {
			log.Printf("failed to scan device version: %v", err)
			continue
		}
		if mgmtIP.Valid {
			dv.mgmtIP = &mgmtIP.String
		}
		devices = append(devices, dv)
	}

	var results []SearchCountResult

	for _, dv := range devices {
		version, err := getVersionByID(db, dv.versionID)
		if err != nil {
			log.Printf("failed to get version %d: %v", dv.versionID, err)
			continue
		}

		content, err := getCachedVersionContent(c.Request.Context(), db, minioClient, version)
		if err != nil {
			log.Printf("failed to get content for device %d: %v", dv.deviceID, err)
			continue
		}

		contentStr := string(content)
		matches := re.FindAllStringIndex(contentStr, -1)
		matchCount := len(matches)

		if matchCount == 0 {
			continue
		}

		lines := strings.Split(contentStr, "\n")
		snippetLines := findSnippetLines(lines, matches, 2)

		results = append(results, SearchCountResult{
			DeviceID:   dv.deviceID,
			VersionID:  dv.versionID,
			Hostname:   dv.hostname,
			MgmtIP:     dv.mgmtIP,
			MatchCount: matchCount,
			Snippets:   snippetLines,
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func findSnippetLines(lines []string, matches [][]int, contextLines int) []SnippetLine {
	// Сначала определяем все номера строк (индексы 0-based), в которых есть совпадение.
	// Так каждая строка с паттерном будет подсвечена, даже если попала в сниппет как контекст другого совпадения.
	matchingLines := make(map[int]bool)
	for _, match := range matches {
		startLine := 0
		pos := 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[0] {
				startLine = i
				break
			}
			pos += lineLen
		}
		endLine := startLine
		pos = 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[1] {
				endLine = i
				break
			}
			pos += lineLen
		}
		for i := startLine; i <= endLine; i++ {
			matchingLines[i] = true
		}
	}

	seenLines := make(map[int]bool)
	var snippetLines []SnippetLine

	for _, match := range matches {
		startLine := 0
		pos := 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[0] {
				startLine = i
				break
			}
			pos += lineLen
		}

		endLine := startLine
		pos = 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[1] {
				endLine = i
				break
			}
			pos += lineLen
		}

		for i := startLine - contextLines; i <= endLine+contextLines; i++ {
			if i < 0 || i >= len(lines) {
				continue
			}
			if seenLines[i] {
				continue
			}
			seenLines[i] = true

			snippetLines = append(snippetLines, SnippetLine{
				Line:  i + 1,
				Text:  lines[i],
				Match: matchingLines[i], // подсвечивать все строки, где есть совпадение с паттерном
			})
		}
	}

	return snippetLines
}

// compilePatterns компилирует шаблоны как буквальный текст (скобки, точки и т.д. не являются regex-метасимволами).
func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, p := range patterns {
		if p == "" {
			continue
		}
		literal := regexp.QuoteMeta(p)
		re, err := regexp.Compile(literal)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

func anyLineMatches(lines []string, res []*regexp.Regexp) bool {
	for _, line := range lines {
		for _, re := range res {
			if re.MatchString(line) {
				return true
			}
		}
	}
	return false
}

type versionPair struct {
	Left  *ConfigVersion
	Right *ConfigVersion
}

func getDeviceVersionPairs(db *sql.DB, deviceID int, fromDate, toDate string) ([]versionPair, error) {
	query := `SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions WHERE device_id = ?`
	args := []interface{}{deviceID}
	if fromDate != "" && len(fromDate) == 10 {
		query += " AND DATE(created_at) >= ?"
		args = append(args, fromDate)
	}
	if toDate != "" && len(toDate) == 10 {
		query += " AND DATE(created_at) <= ?"
		args = append(args, toDate)
	}
	query += " ORDER BY created_at ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*ConfigVersion
	for rows.Next() {
		var v ConfigVersion
		var parentID, chainBaseID sql.NullInt32
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
			&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			v.ParentVersionID = &pid
		}
		if chainBaseID.Valid {
			cid := int(chainBaseID.Int32)
			v.ChainBaseID = &cid
		}
		vCopy := v
		versions = append(versions, &vCopy)
	}

	var pairs []versionPair
	for i := 0; i+1 < len(versions); i++ {
		pairs = append(pairs, versionPair{Left: versions[i], Right: versions[i+1]})
	}
	return pairs, nil
}
