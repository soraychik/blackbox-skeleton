package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"blackbox-api/internal/repository"
	"blackbox-api/internal/service"
	"blackbox-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	db          *sql.DB
	minio       *storage.MinIOImprovedClient
	versionRepo repository.VersionRepository
}

func NewExportHandler(db *sql.DB, minio *storage.MinIOImprovedClient, versionRepo repository.VersionRepository) *ExportHandler {
	return &ExportHandler{
		db:          db,
		minio:       minio,
		versionRepo: versionRepo,
	}
}

func (h *ExportHandler) GetExportConfig(c *gin.Context) {
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

	var versionID int
	var hostname string
	err = h.db.QueryRowContext(c.Request.Context(), `
		SELECT cv.id, d.hostname FROM config_versions cv
		JOIN devices d ON d.id = cv.device_id
		WHERE cv.device_id = ? AND DATE(cv.created_at) = ?
		ORDER BY cv.created_at DESC LIMIT 1`,
		deviceID, dateStr,
	).Scan(&versionID, &hostname)
	if err == sql.ErrNoRows {
		lastDate, _ := h.versionRepo.GetLastDate(c.Request.Context(), deviceID)
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

	version, err := h.versionRepo.GetByID(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	content, err := service.GetCachedVersionContent(c.Request.Context(), h.versionRepo, h.minio, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get config content: %v", err)})
		return
	}

	filename := fmt.Sprintf("config_%s_%s.txt", strings.ReplaceAll(hostname, " ", "_"), dateStr)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "text/plain", content)
}
