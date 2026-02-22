package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"blackbox-api/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
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

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "BlackBox API Web is running...")
	})

	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	router.GET("/devices", getDevices)
	router.GET("/devices/:id", getDeviceByID)
	router.GET("/devices/:id/versions", getDeviceVersions)
	router.GET("/versions", getVersions)
	router.GET("/versions/:id/content", getVersionContent)
	router.GET("/versions/diff/:id1/:id2", getVersionDiff)

	log.Println("API Web server starting on :8080")
	router.Run(":8080")
}

func NewDB() (*sql.DB, error) {
	dbHost := getEnv("DATABASE_HOST", "mysql-db")
	dbPort := getEnv("DATABASE_PORT", "3306")
	dbUser := getEnv("DATABASE_USER", "appuser")
	dbPassword := getEnv("DATABASE_PASSWORD", "apppassword")
	dbName := getEnv("DATABASE_NAME", "blackbox")

	dsn := dbUser + ":" + dbPassword + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?parseTime=true"
	log.Printf("Connecting to MySQL: %s", dsn)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(10)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
	}

	log.Println("Successfully connected to MySQL database")
	return conn, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

func getDevices(c *gin.Context) {
	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

	fromStr := c.Query("from")
	toStr := c.Query("to")

	query := `SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE device_id = ?`

	args := []interface{}{deviceID}

	if fromStr != "" {
		fromID, err := strconv.Atoi(fromStr)
		if err == nil {
			query += " AND id >= ?"
			args = append(args, fromID)
		}
	}
	if toStr != "" {
		toID, err := strconv.Atoi(toStr)
		if err == nil {
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

	c.JSON(http.StatusOK, gin.H{"versions": versions})
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
	defer db.Close()

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

	content, err := reconstructVersionContent(c.Request.Context(), db, minioClient, &version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to reconstruct version: %v", err)})
		return
	}

	computedHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if computedHash != version.VersionHash {
		log.Printf("Warning: hash mismatch for version %d: expected %s, got %s", id, version.VersionHash, computedHash)
	}

	c.String(http.StatusOK, string(content))
}

func reconstructVersionContent(ctx context.Context, db *sql.DB, minioClient *storage.MinIOImprovedClient, version *ConfigVersion) ([]byte, error) {
	log.Printf("DEBUG: reconstructVersionContent called with StorageType=%q, ParentVersionID=%v, StoragePath=%q",
		version.StorageType, version.ParentVersionID, version.StoragePath)

	if version.StorageType == "base" {
		return minioClient.DownloadConfig(ctx, version.StoragePath)
	}

	if version.StorageType == "diff" && version.ParentVersionID != nil {
		parentVersion, err := getVersionByID(db, *version.ParentVersionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent version: %w", err)
		}

		baseContent, err := reconstructVersionContent(ctx, db, minioClient, parentVersion)
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
	Type    string `json:"type"`
	Content string `json:"content"`
	LineNum int    `json:"line_num"`
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
	defer db.Close()

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

	content1, err := reconstructVersionContent(c.Request.Context(), db, minioClient, version1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get content 1: %v", err)})
		return
	}

	content2, err := reconstructVersionContent(c.Request.Context(), db, minioClient, version2)
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
			log.Printf("Failed to cache diff: %v", err)
		}
	}

	diffLines := computeDiffLines(string(content1), string(content2))

	// Отдаём только строки диффа — полные контенты не нужны для отображения и раздувают ответ
	result := DiffResult{
		LeftVersionID:  id1,
		RightVersionID: id2,
		LeftContent:    "",
		RightContent:   "",
		Lines:          diffLines,
	}

	c.JSON(http.StatusOK, result)
}

func computeDiffLines(text1, text2 string) []DiffLine {
	dmp := diffmatchpatch.New()
	lineChar1, lineChar2, lineArray := dmp.DiffLinesToChars(text1, text2)
	diffs := dmp.DiffMain(lineChar1, lineChar2, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	var result []DiffLine
	leftLineNum := 1
	rightLineNum := 1

	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		for _, line := range lines {
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				result = append(result, DiffLine{
					Type:    "unchanged",
					Content: line,
					LineNum: leftLineNum,
				})
				leftLineNum++
				rightLineNum++
			case diffmatchpatch.DiffDelete:
				result = append(result, DiffLine{
					Type:    "removed",
					Content: line,
					LineNum: leftLineNum,
				})
				leftLineNum++
			case diffmatchpatch.DiffInsert:
				result = append(result, DiffLine{
					Type:    "added",
					Content: line,
					LineNum: rightLineNum,
				})
				rightLineNum++
			}
		}
	}

	return result
}
