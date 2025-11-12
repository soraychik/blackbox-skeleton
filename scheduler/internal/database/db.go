package database

import (
	"blackbox-scheduler/internal/models"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	connection *sql.DB
}

// NewDB создаёт новое подключение к БД
func NewDB() (*DB, error) {
	// Добавляем параметр parseTime=true для правильного парсинга дат
	dsn := "appuser:apppassword@tcp(mysql-db:3306)/blackbox?parseTime=true"
	log.Printf("Connecting to MySQL: %s", dsn)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Устанавливаем таймауты
	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(10)

	// Проверяем подключение
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	log.Println("✅ Successfully connected to MySQL database")
	return &DB{connection: conn}, nil
}

// GetOrCreateDevice получает устройство по имени или создаёт новое
func (db *DB) GetOrCreateDevice(name string) (*models.Device, error) {
	var device models.Device

	// Пытаемся найти устройство
	err := db.connection.QueryRow(
		"SELECT id, name, created_at FROM devices WHERE name = ?",
		name,
	).Scan(&device.ID, &device.Name, &device.CreatedAt)

	if err == sql.ErrNoRows {
		// Устройства нет - создаём
		result, err := db.connection.Exec(
			"INSERT INTO devices (name) VALUES (?)",
			name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create device: %v", err)
		}

		id, _ := result.LastInsertId()
		device.ID = int(id)
		device.Name = name
		device.CreatedAt = time.Now()

		log.Printf("Created new device: %s (ID: %d)", name, id)
		return &device, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get device: %v", err)
	}

	log.Printf("Found existing device: %s (ID: %d)", name, device.ID)
	return &device, nil
}

// GetLatestVersion получает последнюю версию конфига для устройства
func (db *DB) GetLatestVersion(deviceID int) (*models.ConfigVersion, error) {
	var version models.ConfigVersion

	err := db.connection.QueryRow(`
        SELECT id, device_id, version_date, file_path, file_hash, created_at 
        FROM config_versions 
        WHERE device_id = ? 
        ORDER BY version_date DESC 
        LIMIT 1`,
		deviceID,
	).Scan(&version.ID, &version.DeviceID, &version.VersionDate, &version.FilePath, &version.FileHash, &version.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // Версий ещё нет - это нормально
	} else if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %v", err)
	}

	return &version, nil
}

// SaveVersion сохраняет новую версию конфига
func (db *DB) SaveVersion(deviceID int, filePath, fileHash string, versionDate time.Time) error {
	_, err := db.connection.Exec(`
        INSERT INTO config_versions (device_id, version_date, file_path, file_hash) 
        VALUES (?, ?, ?, ?)`,
		deviceID, versionDate, filePath, fileHash,
	)

	if err != nil {
		return fmt.Errorf("failed to save version: %v", err)
	}

	log.Printf("Saved new version for device ID %d: %s", deviceID, filePath)
	return nil
}

// Close закрывает подключение к БД
func (db *DB) Close() error {
	return db.connection.Close()
}
