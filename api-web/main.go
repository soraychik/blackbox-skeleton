package main

import (
	"log"
	"net/http"
	"time"

	"blackbox-api/internal/db"
	"blackbox-api/internal/handlers"
	"blackbox-api/internal/repository"
	"blackbox-api/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := db.InitDB(); err != nil {
		log.Fatalf("failed to initialize database pool: %v", err)
	}
	defer db.Pool.Close()

	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		log.Fatalf("failed to connect to minio: %v", err)
	}

	versionRepo := repository.NewVersionRepository(db.Pool)
	deviceRepo := repository.NewDeviceRepository(db.Pool)
	diffRepo := repository.NewDiffRepository(db.Pool)

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
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "BlackBox API Web is running...")
	})

	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	h := handlers.NewHandlers(db.Pool, minioClient, versionRepo, deviceRepo, diffRepo)

	router.GET("/devices", h.Devices.GetDevices)
	router.GET("/devices/:id", h.Devices.GetDeviceByID)
	router.GET("/devices/:id/versions", h.Devices.GetDeviceVersions)
	router.GET("/devices/compare/latest", h.Devices.GetLatestVersionsForDevices)
	router.GET("/dashboard/stats", h.Dashboard.GetDashboardStats)
	router.GET("/versions", h.Versions.GetVersions)
	router.GET("/versions/:id/content", h.Versions.GetVersionContent)
	router.GET("/versions/diff/:id1/:id2", h.Versions.GetVersionDiff)
	router.GET("/diff/date", h.Versions.GetDiffByDate)
	router.GET("/export/config", h.Export.GetExportConfig)
	router.POST("/search/changes", h.Search.PostSearchChanges)
	router.POST("/search/count", h.Search.PostSearchCount)
	router.POST("/scan", handlers.PostTriggerScan)
	router.GET("/scan/status", handlers.GetScanStatus)

	log.Println("api web server starting on :8080")
	router.Run(":8080")
}
