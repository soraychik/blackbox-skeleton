package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
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

	// Или простой вариант CORS:
	// router.Use(cors.Default())

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

	log.Println("API Web server starting on :8080")
	router.Run(":8080")
}

// NewDB создаёт подключение к БД (скопировано из database пакета)
func NewDB() (*sql.DB, error) {
	dsn := "appuser:apppassword@tcp(mysql-db:3306)/blackbox?parseTime=true"
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

	log.Println("✅ Successfully connected to MySQL database")
	return conn, nil
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
