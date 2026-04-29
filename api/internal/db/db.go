package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Pool *sql.DB

var (
	siteMappingsCache map[string]string
	siteMappingsMu    sync.RWMutex
	siteMappingsReady bool
)

func InitDB() error {
	host := getEnv("DATABASE_HOST", "mysql-db")
	port := getEnv("DATABASE_PORT", "3306")
	user := getEnv("DATABASE_USER", "appuser")
	password := getEnv("DATABASE_PASSWORD", "apppassword")
	dbname := getEnv("DATABASE_NAME", "blackbox")

	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?parseTime=true"
	log.Printf("connecting to mysql: %s", dsn)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(30)
	conn.SetMaxIdleConns(30)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return err
	}

	log.Println("successfully connected to mysql database")
	Pool = conn
	if err := EnsureSiteMappingsTable(Pool); err != nil {
		return err
	}
	return nil
}

func EnsureSiteMappingsTable(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS site_mappings (
			code VARCHAR(32) PRIMARY KEY,
			name VARCHAR(128) NOT NULL
		) ENGINE=InnoDB
	`); err != nil {
		return fmt.Errorf("failed to ensure site_mappings table: %w", err)
	}

	defaultMappings := []struct {
		code string
		name string
	}{
		{code: "ekb", name: "Екатеринбург"},
		{code: "ntg", name: "Нижний Тагил"},
		{code: "kur", name: "Каменск-Уральский"},
	}
	for _, m := range defaultMappings {
		if _, err := db.Exec(`
			INSERT INTO site_mappings(code, name)
			VALUES(?, ?)
			ON DUPLICATE KEY UPDATE name = VALUES(name)
		`, m.code, m.name); err != nil {
			return fmt.Errorf("failed to seed site mapping %s: %w", m.code, err)
		}
	}
	return nil
}

func LoadSiteMappings(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT code, name FROM site_mappings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mappings := make(map[string]string)
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		mappings[strings.ToLower(strings.TrimSpace(code))] = name
	}
	return mappings, nil
}

func GetSiteMappings(db *sql.DB) (map[string]string, error) {
	siteMappingsMu.RLock()
	if siteMappingsReady {
		result := make(map[string]string, len(siteMappingsCache))
		for k, v := range siteMappingsCache {
			result[k] = v
		}
		siteMappingsMu.RUnlock()
		return result, nil
	}
	siteMappingsMu.RUnlock()

	siteMappingsMu.Lock()
	defer siteMappingsMu.Unlock()

	if siteMappingsReady {
		result := make(map[string]string, len(siteMappingsCache))
		for k, v := range siteMappingsCache {
			result[k] = v
		}
		return result, nil
	}

	mappings, err := LoadSiteMappings(db)
	if err != nil {
		return nil, err
	}
	siteMappingsCache = mappings
	siteMappingsReady = true
	return mappings, nil
}

func InvalidateSiteMappings() {
	siteMappingsMu.Lock()
	defer siteMappingsMu.Unlock()
	siteMappingsReady = false
	siteMappingsCache = nil
}

func SetSiteMappingsForTest(mappings map[string]string) {
	siteMappingsMu.Lock()
	defer siteMappingsMu.Unlock()
	siteMappingsCache = mappings
	siteMappingsReady = true
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
