//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
)

func getTestDB(t *testing.T) *sql.DB {
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = "3306"
	}
	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("TEST_DB_PASSWORD")
	if password == "" {
		password = "password"
	}
	dbName := os.Getenv("TEST_DB_NAME")
	if dbName == "" {
		dbName = "test_blackbox"
	}

	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbName + "?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skip("skipping integration test: failed to connect to database")
	}
	if err := db.Ping(); err != nil {
		t.Skip("skipping integration test: failed to ping database")
	}
	return db
}

func TestVersionRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()
	repo := NewVersionRepository(db)

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS config_versions (
		id INT AUTO_INCREMENT PRIMARY KEY,
		device_id INT NOT NULL,
		version_hash VARCHAR(64) NOT NULL,
		storage_type VARCHAR(16) NOT NULL,
		storage_path VARCHAR(256) NOT NULL,
		parent_version_id INT,
		chain_base_id INT,
		chain_position INT DEFAULT 0,
		original_size INT DEFAULT 0,
		compressed_size INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.Exec("DELETE FROM config_versions")
	_, err = db.Exec(`INSERT INTO config_versions (device_id, version_hash, storage_type, storage_path) 
		VALUES (1, 'abc123', 'base', 'path/to/config')`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	defer db.Exec("DELETE FROM config_versions")

	version, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if version == nil {
		t.Fatal("expected version, got nil")
	}
	if version.VersionHash != "abc123" {
		t.Errorf("expected hash abc123, got %s", version.VersionHash)
	}
}

func TestVersionRepository_GetPairsByDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()
	repo := NewVersionRepository(db)

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS config_versions (
		id INT AUTO_INCREMENT PRIMARY KEY,
		device_id INT NOT NULL,
		version_hash VARCHAR(64) NOT NULL,
		storage_type VARCHAR(16) NOT NULL,
		storage_path VARCHAR(256) NOT NULL,
		parent_version_id INT,
		chain_base_id INT,
		chain_position INT DEFAULT 0,
		original_size INT DEFAULT 0,
		compressed_size INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.Exec("DELETE FROM config_versions")
	now := time.Now()
	_, err = db.Exec(`INSERT INTO config_versions (device_id, version_hash, storage_type, storage_path, created_at) 
		VALUES (1, 'hash1', 'base', 'path1', ?)`, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	_, err = db.Exec(`INSERT INTO config_versions (device_id, version_hash, storage_type, storage_path, created_at) 
		VALUES (1, 'hash2', 'base', 'path2', ?)`, now)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	defer db.Exec("DELETE FROM config_versions")

	pairs, err := repo.GetPairsByDevice(context.Background(), 1, "", "")
	if err != nil {
		t.Fatalf("GetPairsByDevice failed: %v", err)
	}
	if len(pairs) != 1 {
		t.Errorf("expected 1 pair, got %d", len(pairs))
	}
}

func TestDeviceRepository_GetAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()
	repo := NewDeviceRepository(db)

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS devices (
		id INT AUTO_INCREMENT PRIMARY KEY,
		hostname VARCHAR(128) NOT NULL,
		mgmt_ip VARCHAR(64),
		vendor VARCHAR(64),
		model VARCHAR(64),
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.Exec("DELETE FROM devices")
	_, err = db.Exec(`INSERT INTO devices (hostname, enabled) VALUES ('test-device-1', TRUE)`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	defer db.Exec("DELETE FROM devices")

	devices, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(devices) == 0 {
		t.Error("expected at least one device")
	}
}
