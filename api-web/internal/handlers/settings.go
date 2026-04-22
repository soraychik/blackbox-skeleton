package handlers

import (
	"database/sql"
	"net/http"

	"blackbox-api/internal/models"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	db *sql.DB
}

func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	settings, err := h.loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req models.SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.ConfigSourceType != "" {
		if _, err := h.db.Exec(`
			INSERT INTO system_settings (settings_key, settings_value)
			VALUES ('config_source_type', ?)
			ON DUPLICATE KEY UPDATE settings_value = VALUES(settings_value)
		`, req.ConfigSourceType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config_source_type"})
			return
		}
	}

	if req.ConfigSourcePath != "" {
		if _, err := h.db.Exec(`
			INSERT INTO system_settings (settings_key, settings_value)
			VALUES ('config_source_path', ?)
			ON DUPLICATE KEY UPDATE settings_value = VALUES(settings_value)
		`, req.ConfigSourcePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config_source_path"})
			return
		}
	}

	settings, err := h.loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated settings"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) loadSettings() (*models.SystemSettings, error) {
	settings := &models.SystemSettings{
		ConfigSourceType: "local",
		ConfigSourcePath: "/app/configs",
	}

	var value string
	err := h.db.QueryRow("SELECT settings_value FROM system_settings WHERE settings_key = 'config_source_type'").Scan(&value)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && value != "" {
		settings.ConfigSourceType = value
	}

	err = h.db.QueryRow("SELECT settings_value FROM system_settings WHERE settings_key = 'config_source_path'").Scan(&value)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && value != "" {
		settings.ConfigSourcePath = value
	}

	return settings, nil
}
