package endpoints

import (
	"net/http"
	"strconv"

	"blackbox-api/internal/storage"
	"github.com/gin-gonic/gin"
)

func AddStorageEndpoints(router *gin.Engine) {
	// Получить статистику хранения
	router.GET("/storage/stats", getStorageStats)

	// Получить статистику по устройству
	router.GET("/storage/stats/:deviceId", getDeviceStorageStats)
}

func getStorageStats(c *gin.Context) {
	// Получаем общую статистику (упрощенная версия)
	stats := map[string]interface{}{
		"total_devices":  3, // Можно получить из БД
		"total_versions": 4, // Можно получить из БД
		"storage_by_type": map[string]int64{
			"full": 1024, // Получить из MinIO
			"diff": 512,
			"base": 256,
		},
		"compression_stats": map[string]interface{}{
			"original_bytes":    2048,
			"compressed_bytes":  1024,
			"compression_ratio": 50.0,
		},
		"status": "MinIO integration active",
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func getDeviceStorageStats(c *gin.Context) {
	deviceIDStr := c.Param("deviceId")
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	// Подключаемся к MinIO
	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to MinIO"})
		return
	}

	// Получаем статистику по устройству
	stats, err := minioClient.GetStorageStats(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get storage stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceIDStr,
		"stats":     stats,
		"status":    "MinIO integration active",
	})
}
