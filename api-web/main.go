package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
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
		SELECT cv.id, cv.device_id, d.name, cv.version_date, cv.file_path, cv.file_hash, cv.created_at 
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
		var deviceName, versionDate, filePath, fileHash, createdAt string

		if err := rows.Scan(&id, &deviceID, &deviceName, &versionDate, &filePath, &fileHash, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan version"})
			return
		}

		versions = append(versions, gin.H{
			"id":           id,
			"device_id":    deviceID,
			"device_name":  deviceName,
			"version_date": versionDate,
			"file_path":    filePath,
			"file_hash":    fileHash,
			"created_at":   createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// GetVersionContent возвращает содержимое конфига по ID версии
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

	var filePath string
	err = db.QueryRow("SELECT file_path FROM config_versions WHERE id = ?", id).Scan(&filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Получаем базовый путь к архиву из переменной окружения
	archiveBasePath := getEnv("ARCHIVE_BASE_PATH", "/app/archived_configs")

	// Нормализуем путь
	finalPath := filePath

	// Если путь абсолютный и уже начинается с правильного базового пути, используем как есть
	if filepath.IsAbs(filePath) && strings.HasPrefix(filePath, archiveBasePath) {
		finalPath = filepath.Clean(filePath)
	} else if filepath.IsAbs(filePath) {
		// Если путь абсолютный, но с другим префиксом, извлекаем относительную часть
		if strings.Contains(filePath, "archived_configs") {
			parts := strings.SplitN(filePath, "archived_configs", 2)
			if len(parts) > 1 {
				relPath := strings.TrimPrefix(parts[1], "/")
				finalPath = filepath.Join(archiveBasePath, relPath)
			} else {
				// Если не нашли archived_configs, пробуем использовать как относительный
				finalPath = filepath.Join(archiveBasePath, filepath.Base(filePath))
			}
		} else {
			// Если абсолютный путь без archived_configs, используем имя файла
			finalPath = filepath.Join(archiveBasePath, filepath.Base(filePath))
		}
		finalPath = filepath.Clean(finalPath)
	} else {
		// Если путь относительный, добавляем базовый путь
		finalPath = filepath.Clean(filepath.Join(archiveBasePath, filePath))
	}

	log.Printf("Reading config file: original path='%s', final path='%s', base='%s'", filePath, finalPath, archiveBasePath)

	// Проверяем существование файла
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s (original: %s)", finalPath, filePath)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Config file not found: %s", finalPath)})
		return
	}

	content, err := os.ReadFile(finalPath)
	if err != nil {
		log.Printf("Error reading file %s (original: %s): %v", finalPath, filePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read config file: %v", err)})
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

// GetVersionDiff возвращает diff между двумя версиями
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

	// Получаем пути к файлам
	var filePath1, filePath2 string
	err = db.QueryRow("SELECT file_path FROM config_versions WHERE id = ?", id1).Scan(&filePath1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 1 not found"})
		return
	}

	err = db.QueryRow("SELECT file_path FROM config_versions WHERE id = ?", id2).Scan(&filePath2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version 2 not found"})
		return
	}

	// Получаем базовый путь к архиву
	archiveBasePath := getEnv("ARCHIVE_BASE_PATH", "/app/archived_configs")

	// Нормализуем пути
	normalizePath := func(path string) string {
		// Если путь абсолютный и уже начинается с правильного базового пути, используем как есть
		if filepath.IsAbs(path) && strings.HasPrefix(path, archiveBasePath) {
			return filepath.Clean(path)
		}

		// Если путь абсолютный, но с другим префиксом, извлекаем относительную часть
		if filepath.IsAbs(path) {
			if strings.Contains(path, "archived_configs") {
				parts := strings.SplitN(path, "archived_configs", 2)
				if len(parts) > 1 {
					relPath := strings.TrimPrefix(parts[1], "/")
					return filepath.Clean(filepath.Join(archiveBasePath, relPath))
				}
			}
			// Если не нашли archived_configs, пробуем использовать как относительный
			return filepath.Clean(filepath.Join(archiveBasePath, filepath.Base(path)))
		}

		// Если путь относительный, добавляем базовый путь
		return filepath.Clean(filepath.Join(archiveBasePath, path))
	}

	finalPath1 := normalizePath(filePath1)
	finalPath2 := normalizePath(filePath2)

	log.Printf("Reading diff files: path1='%s' (original: '%s'), path2='%s' (original: '%s')",
		finalPath1, filePath1, finalPath2, filePath2)

	// Проверяем существование файлов
	if _, err := os.Stat(finalPath1); os.IsNotExist(err) {
		log.Printf("File 1 does not exist: %s (original: %s)", finalPath1, filePath1)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Config file 1 not found: %s", finalPath1)})
		return
	}

	if _, err := os.Stat(finalPath2); os.IsNotExist(err) {
		log.Printf("File 2 does not exist: %s (original: %s)", finalPath2, filePath2)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Config file 2 not found: %s", finalPath2)})
		return
	}

	content1, err := os.ReadFile(finalPath1)
	if err != nil {
		log.Printf("Error reading file %s (original: %s): %v", finalPath1, filePath1, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read config file 1: %v", err)})
		return
	}

	content2, err := os.ReadFile(finalPath2)
	if err != nil {
		log.Printf("Error reading file %s (original: %s): %v", finalPath2, filePath2, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read config file 2: %v", err)})
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

// computeDiff вычисляет построчный diff между двумя текстами
func computeDiff(text1, text2 string) []DiffLine {
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")

	// Алгоритм построчного сравнения
	var diff []DiffLine
	i, j := 0, 0
	lineNum1, lineNum2 := 1, 1

	for i < len(lines1) || j < len(lines2) {
		if i >= len(lines1) {
			// Остались только строки из text2 (добавленные)
			diff = append(diff, DiffLine{
				Type:    "added",
				Content: lines2[j],
				LineNum: lineNum2,
			})
			j++
			lineNum2++
		} else if j >= len(lines2) {
			// Остались только строки из text1 (удаленные)
			diff = append(diff, DiffLine{
				Type:    "removed",
				Content: lines1[i],
				LineNum: lineNum1,
			})
			i++
			lineNum1++
		} else if lines1[i] == lines2[j] {
			// Строки одинаковые
			diff = append(diff, DiffLine{
				Type:    "unchanged",
				Content: lines1[i],
				LineNum: lineNum1,
			})
			i++
			j++
			lineNum1++
			lineNum2++
		} else {
			// Строки разные - нужно найти следующее совпадение
			found := false
			lookahead := 5

			for k := 1; k <= lookahead && j+k < len(lines2); k++ {
				if i < len(lines1) && lines1[i] == lines2[j+k] {
					// Найдено совпадение - строки j...j+k-1 добавлены
					for l := 0; l < k; l++ {
						diff = append(diff, DiffLine{
							Type:    "added",
							Content: lines2[j+l],
							LineNum: lineNum2 + l,
						})
					}
					j += k
					lineNum2 += k
					found = true
					break
				}
			}

			if !found {
				// Ищем в обратном направлении
				for k := 1; k <= lookahead && i+k < len(lines1); k++ {
					if j < len(lines2) && lines1[i+k] == lines2[j] {
						// Найдено совпадение - строки i...i+k-1 удалены
						for l := 0; l < k; l++ {
							diff = append(diff, DiffLine{
								Type:    "removed",
								Content: lines1[i+l],
								LineNum: lineNum1 + l,
							})
						}
						i += k
						lineNum1 += k
						found = true
						break
					}
				}
			}

			if !found {
				// Не нашли совпадение - помечаем как измененные
				diff = append(diff, DiffLine{
					Type:    "removed",
					Content: lines1[i],
					LineNum: lineNum1,
				})
				diff = append(diff, DiffLine{
					Type:    "added",
					Content: lines2[j],
					LineNum: lineNum2,
				})
				i++
				j++
				lineNum1++
				lineNum2++
			}
		}
	}

	return diff
}
