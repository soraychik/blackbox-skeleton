package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	db *sql.DB
}

func NewAuditHandler(db *sql.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

type auditEntry struct {
	ID        int64           `json:"id"`
	CreatedAt string          `json:"created_at"`
	Username  string          `json:"username"`
	Role      string          `json:"role"`
	Action    string          `json:"action"`
	Details   json.RawMessage `json:"details"`
	IP        string          `json:"ip_address"`
}

func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	where := []string{"1=1"}
	args := []any{}

	if u := c.Query("username"); u != "" {
		where = append(where, "username = ?")
		args = append(args, u)
	}
	if a := c.Query("action"); a != "" {
		where = append(where, "action = ?")
		args = append(args, a)
	}
	if f := c.Query("from"); f != "" {
		where = append(where, "created_at >= ?")
		args = append(args, f)
	}
	if t := c.Query("to"); t != "" {
		where = append(where, "created_at <= ?")
		args = append(args, t+" 23:59:59.999")
	}

	whereStr := strings.Join(where, " AND ")

	var total int
	h.db.QueryRowContext(c.Request.Context(),
		"SELECT COUNT(*) FROM audit_log WHERE "+whereStr, args...,
	).Scan(&total)

	queryArgs := append(args, limit, offset)
	rows, err := h.db.QueryContext(c.Request.Context(),
		"SELECT id, created_at, username, role, action, details, ip_address FROM audit_log WHERE "+whereStr+
			" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query audit log"})
		return
	}
	defer rows.Close()

	entries := make([]auditEntry, 0)
	for rows.Next() {
		var e auditEntry
		var details sql.NullString
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.Username, &e.Role, &e.Action, &details, &e.IP); err != nil {
			continue
		}
		if details.Valid {
			e.Details = json.RawMessage(details.String)
		} else {
			e.Details = json.RawMessage("{}")
		}
		entries = append(entries, e)
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}
