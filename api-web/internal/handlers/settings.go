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

	updates := map[string]string{}

	if req.ConfigSourceType != "" {
		updates["config_source_type"] = req.ConfigSourceType
	}
	if req.ConfigSourcePath != "" {
		updates["config_source_path"] = req.ConfigSourcePath
	}
	if req.SmbUsername != "" {
		updates["smb_username"] = req.SmbUsername
	}
	if req.SmbPassword != "" {
		updates["smb_password"] = req.SmbPassword
	}
	if req.SmbDomain != "" {
		updates["smb_domain"] = req.SmbDomain
	}

	for key, value := range updates {
		if _, err := h.db.Exec(`
			INSERT INTO system_settings (settings_key, settings_value)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE settings_value = VALUES(settings_value)
		`, key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update " + key})
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
		SmbDomain:        "WORKGROUP",
	}

	rows, err := h.db.Query("SELECT settings_key, settings_value FROM system_settings")
	if err != nil {
		if err == sql.ErrNoRows {
			return settings, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case "config_source_type":
			settings.ConfigSourceType = value
		case "config_source_path":
			settings.ConfigSourcePath = value
		case "smb_username":
			settings.SmbUsername = value
		case "smb_domain":
			settings.SmbDomain = value
		// Пароль намеренно не возвращаем в ответе
		}
	}

	return settings, nil
}