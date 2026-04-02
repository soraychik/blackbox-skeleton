package handlers

import (
	"database/sql"
	"net/http"

	"blackbox-api/internal/models"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	db *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) GetDashboardStats(c *gin.Context) {
	var totalDevices int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalDevices); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count devices"})
		return
	}

	var updatedToday int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM config_versions WHERE DATE(created_at) = CURDATE()").Scan(&updatedToday); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count versions updated today"})
		return
	}

	var devicesWithChanges int
	if err := h.db.QueryRow("SELECT COUNT(DISTINCT device_id) FROM config_versions").Scan(&devicesWithChanges); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count devices with changes"})
		return
	}

	rows, err := h.db.Query(`
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

	var topDevices []models.TopDevice
	for rows.Next() {
		var t models.TopDevice
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
