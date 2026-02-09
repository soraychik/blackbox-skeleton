package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"blackbox-api/internal/endpoints"
	"blackbox-api/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Добавляем CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Главная страница
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "BlackBox API Web is running...")
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Получить все устройства
	router.GET("/devices", getDevices)

	// Получить устройство по ID
	router.GET("/devices/:id", getDeviceByID)

	// Получить все версии конфигов
	router.GET("/versions", getVersions)

	// Получить diff между двумя версиями
	router.GET("/versions/diff/:id1/:id2", getVersionDiff)

	// Получить содержимое конфига по ID версии
	router.GET("/versions/:id/content", getVersionContent)

	// Storage endpoints
	endpoints.AddStorageEndpoints(router)

	log.Println("API Web server starting on :8080")
	router.Run(":8080")
}

// NewDB создаёт подключение к БД (из database пакета)
func NewDB() (*sql.DB, error) {
	// Берем настройки из переменных окружения
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

// GetDevices возвращает список всех устройств
func getDevices(c *gin.Context) {
	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, created_at FROM devices ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query devices"})
		return
	}
	defer rows.Close()

	var devices []gin.H
	for rows.Next() {
		var id int
		var name string
		var createdAt string

		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan device"})
			return
		}

		devices = append(devices, gin.H{
			"id":         id,
			"name":       name,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// GetDeviceByID возвращает устройство по ID
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

	var deviceID int
	var name string
	var createdAt string

	err = db.QueryRow(
		"SELECT id, name, created_at FROM devices WHERE id = ?",
		id,
	).Scan(&deviceID, &name, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device": gin.H{
			"id":         deviceID,
			"name":       name,
			"created_at": createdAt,
		},
	})
}

// GetVersions возвращает все версии конфигураций
func getVersions(c *gin.Context) {
	db, err := NewDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT cv.id, cv.device_id, d.name, cv.version_date, cv.file_path, cv.file_hash, cv.storage_type, cv.minio_object_name, cv.diff_size_bytes, cv.created_at 
		FROM config_versions cv
		JOIN devices d ON cv.device_id = d.id
		ORDER BY cv.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query versions"})
		return
	}
	defer rows.Close()

	var versions []gin.H
	for rows.Next() {
		var id, deviceID int
		var deviceName, versionDate, filePath, fileHash, storageType, minioObjectName string
		var diffSizeBytes int
		var createdAt string

		if err := rows.Scan(&id, &deviceID, &deviceName, &versionDate, &filePath, &fileHash, &storageType, &minioObjectName, &diffSizeBytes, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan version"})
			return
		}

		versions = append(versions, gin.H{
			"id":                id,
			"device_id":         deviceID,
			"device_name":       deviceName,
			"version_date":      versionDate,
			"file_path":         filePath,
			"file_hash":         fileHash,
			"storage_type":      storageType,
			"minio_object_name": minioObjectName,
			"diff_size_bytes":   diffSizeBytes,
			"created_at":        createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// GetVersionContent возвращает содержимое конфига по ID версии из MinIO
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

	var minioObjectName string
	var filePath string
	err = db.QueryRow("SELECT minio_object_name, file_path FROM config_versions WHERE id = ?", id).Scan(&minioObjectName, &filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Если minio_object_name пуст, пробуем конвертировать старый путь
	if minioObjectName == "" && filePath != "" {
		minioObjectName = strings.TrimPrefix(filePath, "configs/")
	}

	// Если все еще пустой, возвращаем ошибку
	if minioObjectName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No MinIO object name found for this version"})
		return
	}

	// Подключаемся к MinIO
	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	content, err := minioClient.DownloadConfig(c.Request.Context(), minioObjectName)
	if err != nil {
		log.Printf("Failed to download config %s from MinIO: %v", minioObjectName, err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Config file not found in MinIO: %s", minioObjectName)})
		return
	}

	c.String(http.StatusOK, string(content))
}

// DiffLine представляет одну строку в diff
type DiffLine struct {
	Type    string `json:"type"`     // "added", "removed", "unchanged"
	Content string `json:"content"`  // содержимое строки
	LineNum int    `json:"line_num"` // номер строки (для левой или правой версии)
}

// DiffResult представляет результат сравнения
type DiffResult struct {
	LeftVersionID  int        `json:"left_version_id"`
	RightVersionID int        `json:"right_version_id"`
	LeftContent    string     `json:"left_content"`
	RightContent   string     `json:"right_content"`
	Lines          []DiffLine `json:"lines"`
}

// GetVersionDiff возвращает diff между двумя версиями из MinIO
func getVersionDiff(c *gin.Context) {
	id1Param := c.Param("id1")
	id2Param := c.Param("id2")

	id1, err := strconv.Atoi(id1Param)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID 1"})
		return
	}

	id2, err := strconv.Atoi(id2Param)
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

	// Получаем имена объектов в MinIO
	var minioObjectName1, minioObjectName2 string
	var filePath1, filePath2 string

	err = db.QueryRow("SELECT minio_object_name, file_path FROM config_versions WHERE id = ?", id1).Scan(&minioObjectName1, &filePath1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 1 not found"})
		return
	}

	err = db.QueryRow("SELECT minio_object_name, file_path FROM config_versions WHERE id = ?", id2).Scan(&minioObjectName2, &filePath2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 2 not found"})
		return
	}

	// Если minio_object_name пуст, пробуем конвертировать старый путь
	if minioObjectName1 == "" && filePath1 != "" {
		minioObjectName1 = strings.TrimPrefix(filePath1, "configs/")
	}
	if minioObjectName2 == "" && filePath2 != "" {
		minioObjectName2 = strings.TrimPrefix(filePath2, "configs/")
	}

	// Если все еще пустые, возвращаем ошибку
	if minioObjectName1 == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No MinIO object name found for version 1"})
		return
	}
	if minioObjectName2 == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No MinIO object name found for version 2"})
		return
	}

	// Подключаемся к MinIO
	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to MinIO: %v", err)})
		return
	}

	log.Printf("Reading diff from MinIO: object1='%s', object2='%s'", minioObjectName1, minioObjectName2)

	content1, err := minioClient.DownloadConfig(c.Request.Context(), minioObjectName1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Failed to download config 1 from MinIO: %v", err)})
		return
	}

	content2, err := minioClient.DownloadConfig(c.Request.Context(), minioObjectName2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Failed to download config 2 from MinIO: %v", err)})
		return
	}

	// Вычисляем diff
	diff := computeDiff(string(content1), string(content2))

	result := DiffResult{
		LeftVersionID:  id1,
		RightVersionID: id2,
		LeftContent:    string(content1),
		RightContent:   string(content2),
		Lines:          diff,
	}

	c.JSON(http.StatusOK, result)
}

// computeDiff вычисляет построчный diff между двумя текстами с использованием Google's optimized diff-match-patch
func computeDiff(text1, text2 string) []DiffLine {
	dmp := diffmatchpatch.New()

	// Используем line mode для построчного сравнения
	charData1, charData2, lineArray := dmp.DiffLinesToChars(text1, text2)
	diffs := dmp.DiffMain(charData1, charData2, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	// Конвертируем в наш формат DiffLine
	var result []DiffLine
	leftLineNum := 1
	rightLineNum := 1

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")

		// Удаляем пустую строку в конце если есть
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		for _, line := range lines {
			if line == "" {
				continue
			}

			var diffType string
			var lineNum int

			switch diff.Type {
			case diffmatchpatch.DiffInsert:
				diffType = "added"
				lineNum = rightLineNum
				rightLineNum++
			case diffmatchpatch.DiffDelete:
				diffType = "removed"
				lineNum = leftLineNum
				leftLineNum++
			case diffmatchpatch.DiffEqual:
				diffType = "unchanged"
				lineNum = leftLineNum // для unchanged номера одинаковые
				leftLineNum++
				rightLineNum++
			}

			result = append(result, DiffLine{
				Type:    diffType,
				Content: line,
				LineNum: lineNum,
			})
		}
	}

	return result
}
