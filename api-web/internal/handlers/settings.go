package handlers

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	if req.ScanIntervalSeconds >= 5 {
		updates["scan_interval_seconds"] = strconv.Itoa(req.ScanIntervalSeconds)
	}

	if req.LDAPEnabled {
		updates["ldap_enabled"] = "true"
	} else {
		updates["ldap_enabled"] = "false"
	}
	if req.LDAPUrl != "" {
		updates["ldap_url"] = req.LDAPUrl
	}
	if req.LDAPBindDN != "" {
		updates["ldap_bind_dn"] = req.LDAPBindDN
	}
	if req.LDAPBindPassword != "" {
		updates["ldap_bind_password"] = req.LDAPBindPassword
	}
	if req.LDAPUserBase != "" {
		updates["ldap_user_base"] = req.LDAPUserBase
	}
	if req.LDAPUserFilter != "" {
		updates["ldap_user_filter"] = req.LDAPUserFilter
	}
	if req.LDAPRoleAdmin != "" {
		updates["ldap_role_admin"] = req.LDAPRoleAdmin
	}
	if req.LDAPRoleEngineer != "" {
		updates["ldap_role_engineer"] = req.LDAPRoleEngineer
	}
	if req.LDAPRoleOperator != "" {
		updates["ldap_role_operator"] = req.LDAPRoleOperator
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

	notifySchedulerReload()

	settings, err := h.loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated settings"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func notifySchedulerReload() {
	url := strings.TrimSuffix(getSchedulerURL(), "/") + "/reload"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		log.Printf("reload notify: failed to create request: %v", err)
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("reload notify: scheduler unreachable: %v", err)
		return
	}
	resp.Body.Close()
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
		case "scan_interval_seconds":
			if n, err := strconv.Atoi(value); err == nil {
				settings.ScanIntervalSeconds = n
			}
		case "ldap_enabled":
			settings.LDAPEnabled = value == "true"
		case "ldap_url":
			settings.LDAPUrl = value
		case "ldap_bind_dn":
			settings.LDAPBindDN = value
		case "ldap_user_base":
			settings.LDAPUserBase = value
		case "ldap_user_filter":
			settings.LDAPUserFilter = value
		case "ldap_role_admin":
			settings.LDAPRoleAdmin = value
		case "ldap_role_engineer":
			settings.LDAPRoleEngineer = value
		case "ldap_role_operator":
			settings.LDAPRoleOperator = value
		// smb_password и ldap_bind_password намеренно не возвращаем в ответе
		}
	}

	return settings, nil
}