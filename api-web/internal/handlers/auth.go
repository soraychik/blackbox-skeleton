package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"blackbox-api/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type loginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Введите логин и пароль"})
		return
	}

	// Local admin fallback — работает даже если LDAP не настроен
	if adminLogin := os.Getenv("ADMIN_LOGIN"); adminLogin != "" {
		if req.Login == adminLogin && req.Password == os.Getenv("ADMIN_PASSWORD") {
			h.issueAndRespond(c, req.Login, "admin", req.Remember)
			return
		}
	}

	cfg, err := h.loadLDAPConfig()
	if err != nil || !cfg.Enabled {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}

	role, err := ldapAuthenticate(cfg, req.Login, req.Password)
	if err != nil {
		log.Printf("ldap auth failed for %s: %v", req.Login, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}

	h.issueAndRespond(c, req.Login, role, req.Remember)
}

func (h *AuthHandler) issueAndRespond(c *gin.Context, username, role string, remember bool) {
	ttl := 24 * time.Hour
	if remember {
		ttl = 7 * 24 * time.Hour
	}
	token, err := auth.IssueToken(username, role, ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"login": username, "role": role},
	})
}

type ldapCfg struct {
	Enabled      bool
	URL          string
	BindDN       string
	BindPassword string
	UserBase     string
	UserFilter   string
	RoleAdmin    string
	RoleEngineer string
	RoleOperator string
}

func (h *AuthHandler) loadLDAPConfig() (*ldapCfg, error) {
	rows, err := h.db.Query("SELECT settings_key, settings_value FROM system_settings WHERE settings_key LIKE 'ldap_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cfg := &ldapCfg{UserFilter: "(sAMAccountName=%s)"}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		switch k {
		case "ldap_enabled":
			cfg.Enabled = v == "true"
		case "ldap_url":
			cfg.URL = v
		case "ldap_bind_dn":
			cfg.BindDN = v
		case "ldap_bind_password":
			cfg.BindPassword = v
		case "ldap_user_base":
			cfg.UserBase = v
		case "ldap_user_filter":
			if v != "" {
				cfg.UserFilter = v
			}
		case "ldap_role_admin":
			cfg.RoleAdmin = v
		case "ldap_role_engineer":
			cfg.RoleEngineer = v
		case "ldap_role_operator":
			cfg.RoleOperator = v
		}
	}
	return cfg, nil
}

func ldapAuthenticate(cfg *ldapCfg, username, password string) (string, error) {
	conn, err := ldap.DialURL(cfg.URL)
	if err != nil {
		return "", fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return "", fmt.Errorf("service bind: %w", err)
	}

	filter := fmt.Sprintf(cfg.UserFilter, ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		cfg.UserBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"dn", "memberOf"},
		nil,
	)
	result, err := conn.Search(searchReq)
	if err != nil {
		return "", fmt.Errorf("search: %w", err)
	}
	if len(result.Entries) == 0 {
		return "", fmt.Errorf("user not found")
	}

	userDN := result.Entries[0].DN
	memberOf := result.Entries[0].GetAttributeValues("memberOf")

	if err := conn.Bind(userDN, password); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	return mapRole(cfg, memberOf)
}

func mapRole(cfg *ldapCfg, memberOf []string) (string, error) {
	for _, g := range memberOf {
		if cfg.RoleAdmin != "" && strings.EqualFold(g, cfg.RoleAdmin) {
			return "admin", nil
		}
	}
	for _, g := range memberOf {
		if cfg.RoleEngineer != "" && strings.EqualFold(g, cfg.RoleEngineer) {
			return "engineer", nil
		}
	}
	for _, g := range memberOf {
		if cfg.RoleOperator != "" && strings.EqualFold(g, cfg.RoleOperator) {
			return "operator", nil
		}
	}
	return "", fmt.Errorf("user has no valid BlackBox role")
}
