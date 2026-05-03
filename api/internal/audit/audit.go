package audit

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
)

type Logger struct {
	db *sql.DB
}

func New(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// Log writes an audit entry asynchronously so it never blocks the response.
func (l *Logger) Log(username, role, action string, details map[string]any, ip string) {
	go func() {
		data, _ := json.Marshal(details)
		_, err := l.db.Exec(
			`INSERT INTO audit_log (username, role, action, details, ip_address) VALUES (?, ?, ?, ?, ?)`,
			username, role, action, string(data), ip,
		)
		if err != nil {
			log.Printf("audit: %v", err)
		}
	}()
}

func FromGin(c *gin.Context) (username, role, ip string) {
	if u, ok := c.Get("username"); ok {
		username, _ = u.(string)
	}
	if r, ok := c.Get("role"); ok {
		role, _ = r.(string)
	}
	ip = c.ClientIP()
	return
}
