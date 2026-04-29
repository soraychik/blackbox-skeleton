package main

import (
	"log"
	"net/http"
	"time"

	"blackbox-api/internal/audit"
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
	auditLog := audit.New(db.Pool)

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

	authHandler := handlers.NewAuthHandler(db.Pool, auditLog)
	router.POST("/auth/login", authHandler.Login)

	h := handlers.NewHandlers(db.Pool, minioClient, versionRepo, deviceRepo, diffRepo, auditLog)

	adminEngineer := auth.RequireRole("admin", "engineer")
	adminOnly := auth.RequireRole("admin")

	protected := router.Group("/")
	protected.Use(auth.Middleware())
	{
		// all authenticated roles
		protected.GET("/devices", h.Devices.GetDevices)
		protected.GET("/devices/:id", h.Devices.GetDeviceByID)
		protected.GET("/devices/:id/versions", h.Devices.GetDeviceVersions)
		protected.GET("/dashboard/stats", h.Dashboard.GetDashboardStats)
		protected.GET("/versions", h.Versions.GetVersions)
		protected.GET("/versions/:id/content", h.Versions.GetVersionContent)
		protected.GET("/export/config", h.Export.GetExportConfig)
		protected.GET("/scan/status", handlers.GetScanStatus)

		// admin + engineer
		protected.GET("/devices/compare/latest", adminEngineer, h.Devices.GetLatestVersionsForDevices)
		protected.GET("/versions/diff/:id1/:id2", adminEngineer, h.Versions.GetVersionDiff)
		protected.GET("/diff/date", adminEngineer, h.Versions.GetDiffByDate)
		protected.POST("/search/changes", adminEngineer, h.Search.PostSearchChanges)
		protected.POST("/search/count", adminEngineer, h.Search.PostSearchCount)

		// admin only
		protected.POST("/scan", adminOnly, h.Scan.PostTriggerScan)
		protected.GET("/settings", adminOnly, h.Settings.GetSettings)
		protected.PUT("/settings", adminOnly, h.Settings.UpdateSettings)
		protected.GET("/audit", adminOnly, h.Audit.GetAuditLog)
	}

	log.Println("api web server starting on :8080")
	router.Run(":8080")
}
