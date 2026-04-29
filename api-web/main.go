package main

import (
	"log"
	"net/http"
	"time"

	"blackbox-api/internal/auth"
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

	authHandler := handlers.NewAuthHandler(db.Pool)
	router.POST("/auth/login", authHandler.Login)

	h := handlers.NewHandlers(db.Pool, minioClient, versionRepo, deviceRepo, diffRepo)

	protected := router.Group("/")
	protected.Use(auth.Middleware())
	{
		protected.GET("/devices", h.Devices.GetDevices)
		protected.GET("/devices/:id", h.Devices.GetDeviceByID)
		protected.GET("/devices/:id/versions", h.Devices.GetDeviceVersions)
		protected.GET("/devices/compare/latest", h.Devices.GetLatestVersionsForDevices)
		protected.GET("/dashboard/stats", h.Dashboard.GetDashboardStats)
		protected.GET("/versions", h.Versions.GetVersions)
		protected.GET("/versions/:id/content", h.Versions.GetVersionContent)
		protected.GET("/versions/diff/:id1/:id2", h.Versions.GetVersionDiff)
		protected.GET("/diff/date", h.Versions.GetDiffByDate)
		protected.GET("/export/config", h.Export.GetExportConfig)
		protected.POST("/search/changes", h.Search.PostSearchChanges)
		protected.POST("/search/count", h.Search.PostSearchCount)
		protected.POST("/scan", handlers.PostTriggerScan)
		protected.GET("/scan/status", handlers.GetScanStatus)
		protected.GET("/settings", h.Settings.GetSettings)
		protected.PUT("/settings", h.Settings.UpdateSettings)
	}

	log.Println("api web server starting on :8080")
	router.Run(":8080")
}
