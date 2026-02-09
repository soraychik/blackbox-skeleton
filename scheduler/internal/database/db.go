package database

import (
	"blackbox-scheduler/internal/models"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	connection *sql.DB
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// NewDB создаёт новое подключение к БД
func NewDB() (*DB, error) {
	// Берем настройки из переменных окружения
	dbHost := getEnv("DATABASE_HOST", "mysql-db")
	dbPort := getEnv("DATABASE_PORT", "3306")
	dbUser := getEnv("DATABASE_USER", "appuser")
	dbPassword := getEnv("DATABASE_PASSWORD", "apppassword")
	dbName := getEnv("DATABASE_NAME", "blackbox")

	// Формируем DSN строку из переменных окружения
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)

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

	log.Println("Successfully connected to MySQL database")
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
		return nil, nil // Версий ещё нет
	} else if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %v", err)
	}

	return &version, nil
}

// SaveVersion сохраняет новую версию конфига (legacy)
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

// SaveFullVersion сохраняет полную версию конфигурации
func (db *DB) SaveFullVersion(deviceID int, minioObjectName, fileHash string, versionDate time.Time, storageType string, originalSize, compressedSize int64) (*models.ConfigVersion, error) {
	result, err := db.connection.Exec(`
        INSERT INTO config_versions (device_id, version_date, file_path, file_hash, storage_type, minio_object_name, diff_size_bytes) 
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		deviceID, versionDate, minioObjectName, fileHash, storageType, minioObjectName, int(compressedSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save full version: %v", err)
	}

	id, _ := result.LastInsertId()

	// Сохраняем информацию о снапшоте
	_, err = db.connection.Exec(`
        INSERT INTO storage_snapshots (device_id, version_id, snapshot_type, minio_object_name, file_size_bytes, compressed_size_bytes) 
        VALUES (?, ?, ?, ?, ?, ?)`,
		deviceID, id, storageType, minioObjectName, originalSize, compressedSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot info: %v", err)
	}

	version := &models.ConfigVersion{
		ID:              int(id),
		DeviceID:        deviceID,
		VersionDate:     versionDate,
		FilePath:        minioObjectName,
		FileHash:        fileHash,
		StorageType:     storageType,
		ParentVersionID: nil,
		MinioObjectName: minioObjectName,
		DiffSizeBytes:   int(compressedSize),
		CreatedAt:       time.Now(),
	}

	log.Printf("Saved new full version for device ID %d: %s (storage: %s)", deviceID, minioObjectName, storageType)
	return version, nil
}

// SaveDiffVersion сохраняет diff версию конфигурации
func (db *DB) SaveDiffVersion(deviceID int, minioObjectName, fileHash string, versionDate time.Time, parentVersionID int, diffSizeBytes int64) (*models.ConfigVersion, error) {
	result, err := db.connection.Exec(`
        INSERT INTO config_versions (device_id, version_date, file_path, file_hash, storage_type, parent_version_id, minio_object_name, diff_size_bytes) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		deviceID, versionDate, minioObjectName, fileHash, "diff", parentVersionID, minioObjectName, diffSizeBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save diff version: %v", err)
	}

	id, _ := result.LastInsertId()

	version := &models.ConfigVersion{
		ID:              int(id),
		DeviceID:        deviceID,
		VersionDate:     versionDate,
		FilePath:        minioObjectName,
		FileHash:        fileHash,
		StorageType:     "diff",
		ParentVersionID: &parentVersionID,
		MinioObjectName: minioObjectName,
		DiffSizeBytes:   int(diffSizeBytes),
		CreatedAt:       time.Now(),
	}

	log.Printf("Saved new diff version for device ID %d: %s (parent: %d)", deviceID, minioObjectName, parentVersionID)
	return version, nil
}

// GetVersionByID получает версию по ID
func (db *DB) GetVersionByID(versionID int) (*models.ConfigVersion, error) {
	var version models.ConfigVersion
	var parentVersionID sql.NullInt32

	err := db.connection.QueryRow(`
        SELECT id, device_id, version_date, file_path, file_hash, storage_type, parent_version_id, minio_object_name, diff_size_bytes, created_at 
        FROM config_versions 
        WHERE id = ?`,
		versionID,
	).Scan(&version.ID, &version.DeviceID, &version.VersionDate, &version.FilePath, &version.FileHash,
		&version.StorageType, &parentVersionID, &version.MinioObjectName, &version.DiffSizeBytes, &version.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get version by ID: %v", err)
	}

	if parentVersionID.Valid {
		pid := int(parentVersionID.Int32)
		version.ParentVersionID = &pid
	}

	return &version, nil
}

// GetAllVersionsForMigration получает все версии для миграции
func (db *DB) GetAllVersionsForMigration() ([]models.ConfigVersion, error) {
	rows, err := db.connection.Query(`
		SELECT id, device_id, version_date, file_path, file_hash, created_at 
		FROM config_versions 
		WHERE storage_type IS NULL OR storage_type = 'full' 
		ORDER BY device_id, version_date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions for migration: %v", err)
	}
	defer rows.Close()

	var versions []models.ConfigVersion
	for rows.Next() {
		var version models.ConfigVersion

		err := rows.Scan(&version.ID, &version.DeviceID, &version.VersionDate,
			&version.FilePath, &version.FileHash, &version.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %v", err)
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// UpdateVersionForMigration обновляет версию после миграции
func (db *DB) UpdateVersionForMigration(versionID int, minioObjectName, storageType string, originalSize, compressedSize int64) error {
	_, err := db.connection.Exec(`
		UPDATE config_versions 
		SET storage_type = ?, minio_object_name = ?, diff_size_bytes = ?
		WHERE id = ?`,
		storageType, minioObjectName, compressedSize, versionID,
	)
	if err != nil {
		return fmt.Errorf("failed to update version for migration: %v", err)
	}

	return nil
}

// CreateStorageSnapshot создает запись в storage_snapshots
func (db *DB) CreateStorageSnapshot(deviceID, versionID int, snapshotType, minioObjectName string, fileSize, compressedSize int64) error {
	_, err := db.connection.Exec(`
		INSERT INTO storage_snapshots (device_id, version_id, snapshot_type, minio_object_name, file_size_bytes, compressed_size_bytes) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		deviceID, versionID, snapshotType, minioObjectName, fileSize, compressedSize,
	)
	if err != nil {
		return fmt.Errorf("failed to create storage snapshot: %v", err)
	}

	return nil
}

// Закрывает подключение к БД
func (db *DB) Close() error {
	return db.connection.Close()
}
