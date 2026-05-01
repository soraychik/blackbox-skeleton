package database

import (
	"blackbox-scheduler/internal/models"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	connection *sql.DB
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewDB() (*DB, error) {
	dbHost := getEnv("DATABASE_HOST", "mysql-db")
	dbPort := getEnv("DATABASE_PORT", "3306")
	dbUser := getEnv("DATABASE_USER", "appuser")
	dbPassword := getEnv("DATABASE_PASSWORD", "apppassword")
	dbName := getEnv("DATABASE_NAME", "blackbox")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	log.Printf("connecting to mysql: %s", dsn)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(10)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("successfully connected to mysql database")
	return &DB{connection: conn}, nil
}

// FileState хранит последнее известное состояние файла на диске
type FileState struct {
	Hostname string
	Size     int64
	ModTime  time.Time
}

// truncateToMillis обрезает время до миллисекунд.
// os.Stat возвращает наносекунды, MySQL DATETIME(3) хранит миллисекунды.
// Без обрезки time.Equal возвращает false даже при одинаковом времени.
func truncateToMillis(t time.Time) time.Time {
	return t.Truncate(time.Millisecond)
}

// GetFileState возвращает последнее сохранённое состояние файла по hostname.
// Возвращает nil, nil если запись не найдена.
func (db *DB) GetFileState(hostname string) (*FileState, error) {
	var fs FileState
	var modTime time.Time
	err := db.connection.QueryRow(`
		SELECT hostname, file_size, file_mtime
		FROM device_file_state
		WHERE hostname = ?`, hostname,
	).Scan(&fs.Hostname, &fs.Size, &modTime)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetFileState: %w", err)
	}
	fs.ModTime = truncateToMillis(modTime)
	return &fs, nil
}

// UpsertFileState сохраняет или обновляет состояние файла.
func (db *DB) UpsertFileState(hostname string, size int64, modTime time.Time) error {
	_, err := db.connection.Exec(`
		INSERT INTO device_file_state (hostname, file_size, file_mtime)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE file_size = VALUES(file_size), file_mtime = VALUES(file_mtime)`,
		hostname, size, truncateToMillis(modTime),
	)
	if err != nil {
		return fmt.Errorf("UpsertFileState: %w", err)
	}
	return nil
}

// VersionCandidate — результат обработки одного файла, готовый к записи в БД.
type VersionCandidate struct {
	DeviceID        int
	VersionHash     string
	StorageType     string
	StoragePath     string
	ParentVersionID *int
	ChainBaseID     *int
	ChainPosition   int
	OriginalSize    uint32
	CompressedSize  uint32
	// Для обновления device_file_state после успешной записи
	Hostname string
	FileSize int64
	FileTime time.Time
}

// SaveVersionsBatch сохраняет несколько версий одной транзакцией.
// Каждая версия вставляется отдельным INSERT (MySQL не поддерживает
// multi-row INSERT с LAST_INSERT_ID для каждой строки), но все в одной tx —
// это даёт значительный выигрыш по сравнению с N отдельных транзакций.
// После успешной записи обновляется device_file_state для каждого устройства.
func (db *DB) SaveVersionsBatch(candidates []VersionCandidate) error {
	if len(candidates) == 0 {
		return nil
	}

	tx, err := db.connection.Begin()
	if err != nil {
		return fmt.Errorf("SaveVersionsBatch begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO config_versions
		(device_id, version_hash, storage_type, storage_path,
		 parent_version_id, chain_base_id, chain_position, original_size, compressed_size)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("SaveVersionsBatch prepare: %w", err)
	}
	defer stmt.Close()

	for _, c := range candidates {
		var parentID, baseID interface{}
		if c.ParentVersionID != nil {
			parentID = *c.ParentVersionID
		}
		if c.ChainBaseID != nil {
			baseID = *c.ChainBaseID
		}
		if _, err := stmt.Exec(
			c.DeviceID, c.VersionHash, c.StorageType, c.StoragePath,
			parentID, baseID, c.ChainPosition, c.OriginalSize, c.CompressedSize,
		); err != nil {
			return fmt.Errorf("SaveVersionsBatch insert device %d: %w", c.DeviceID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SaveVersionsBatch commit: %w", err)
	}

	// Обновляем file state после успешной записи всех версий
	for _, c := range candidates {
		if c.Hostname != "" {
			if err := db.UpsertFileState(c.Hostname, c.FileSize, c.FileTime); err != nil {
				log.Printf("warning: failed to upsert file state for %s: %v", c.Hostname, err)
			}
		}
	}

	log.Printf("batch saved %d versions", len(candidates))
	return nil
}

// ResolveHostnames возвращает map hostname -> device_id для списка хостов,
// создавая устройства которых ещё нет. Один запрос вместо N.
func (db *DB) ResolveHostnames(hostnames []string) (map[string]int, error) {
	if len(hostnames) == 0 {
		return map[string]int{}, nil
	}

	// Сначала получаем все уже существующие
	placeholders := strings.Repeat("?,", len(hostnames))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]interface{}, len(hostnames))
	for i, h := range hostnames {
		args[i] = h
	}

	rows, err := db.connection.Query(
		"SELECT id, hostname FROM devices WHERE hostname IN ("+placeholders+")", args...,
	)
	if err != nil {
		return nil, fmt.Errorf("ResolveHostnames query: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int, len(hostnames))
	for rows.Next() {
		var id int
		var hostname string
		if err := rows.Scan(&id, &hostname); err != nil {
			return nil, err
		}
		result[hostname] = id
	}

	// Создаём отсутствующие одним батчем
	var missing []string
	for _, h := range hostnames {
		if _, ok := result[h]; !ok {
			missing = append(missing, h)
		}
	}

	if len(missing) > 0 {
		placeholders = strings.Repeat("(?),", len(missing))
		placeholders = placeholders[:len(placeholders)-1]
		insertArgs := make([]interface{}, len(missing))
		for i, h := range missing {
			insertArgs[i] = h
		}
		if _, err := db.connection.Exec(
			"INSERT IGNORE INTO devices (hostname) VALUES "+placeholders, insertArgs...,
		); err != nil {
			return nil, fmt.Errorf("ResolveHostnames insert: %w", err)
		}

		// Перечитываем созданные
		rows2, err := db.connection.Query(
			"SELECT id, hostname FROM devices WHERE hostname IN ("+placeholders+")", insertArgs...,
		)
		if err != nil {
			return nil, fmt.Errorf("ResolveHostnames re-query: %w", err)
		}
		defer rows2.Close()
		for rows2.Next() {
			var id int
			var hostname string
			if err := rows2.Scan(&id, &hostname); err != nil {
				return nil, err
			}
			result[hostname] = id
		}
	}

	return result, nil
}

func (db *DB) GetOrCreateDevice(hostname string) (*models.Device, error) {
	var device models.Device

	err := db.connection.QueryRow(
		"SELECT id, hostname, created_at FROM devices WHERE hostname = ?",
		hostname,
	).Scan(&device.ID, &device.Hostname, &device.CreatedAt)

	if err == sql.ErrNoRows {
		result, err := db.connection.Exec(
			"INSERT INTO devices (hostname) VALUES (?)",
			hostname,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create device: %w", err)
		}

		id, _ := result.LastInsertId()
		device.ID = int(id)
		device.Hostname = hostname
		device.CreatedAt = time.Now()
		device.Enabled = true

		log.Printf("created new device: %s (id: %d)", hostname, id)
		return &device, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	log.Printf("found existing device: %s (id: %d)", hostname, device.ID)
	return &device, nil
}

func (db *DB) GetDeviceByID(id int) (*models.Device, error) {
	var device models.Device

	err := db.connection.QueryRow(
		"SELECT id, hostname, mgmt_ip, vendor, model, tags, enabled, created_at FROM devices WHERE id = ?",
		id,
	).Scan(&device.ID, &device.Hostname, &device.MgmtIP, &device.Vendor, &device.Model, &device.Tags, &device.Enabled, &device.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return &device, nil
}

func (db *DB) GetAllDevices() ([]models.Device, error) {
	rows, err := db.connection.Query(
		"SELECT id, hostname, mgmt_ip, vendor, model, tags, enabled, created_at FROM devices ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var device models.Device
		if err := rows.Scan(&device.ID, &device.Hostname, &device.MgmtIP, &device.Vendor, &device.Model, &device.Tags, &device.Enabled, &device.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

func (db *DB) GetLatestVersion(deviceID int) (*models.ConfigVersion, error) {
	var version models.ConfigVersion
	var parentVersionID, chainBaseID sql.NullInt32

	err := db.connection.QueryRow(`
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE device_id = ? 
		ORDER BY id DESC 
		LIMIT 1`,
		deviceID,
	).Scan(&version.ID, &version.DeviceID, &version.VersionHash, &version.StorageType, &version.StoragePath,
		&parentVersionID, &chainBaseID, &version.ChainPosition, &version.OriginalSize, &version.CompressedSize, &version.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	if parentVersionID.Valid {
		pid := int(parentVersionID.Int32)
		version.ParentVersionID = &pid
	}
	if chainBaseID.Valid {
		cid := int(chainBaseID.Int32)
		version.ChainBaseID = &cid
	}

	return &version, nil
}

func (db *DB) GetVersionByID(versionID int) (*models.ConfigVersion, error) {
	var version models.ConfigVersion
	var parentVersionID, chainBaseID sql.NullInt32

	err := db.connection.QueryRow(`
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE id = ?`,
		versionID,
	).Scan(&version.ID, &version.DeviceID, &version.VersionHash, &version.StorageType, &version.StoragePath,
		&parentVersionID, &chainBaseID, &version.ChainPosition, &version.OriginalSize, &version.CompressedSize, &version.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get version by id: %w", err)
	}

	if parentVersionID.Valid {
		pid := int(parentVersionID.Int32)
		version.ParentVersionID = &pid
	}
	if chainBaseID.Valid {
		cid := int(chainBaseID.Int32)
		version.ChainBaseID = &cid
	}

	return &version, nil
}

func (db *DB) GetVersionsInChain(chainBaseID int) ([]models.ConfigVersion, error) {
	rows, err := db.connection.Query(`
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE chain_base_id = ? OR id = ?
		ORDER BY chain_position ASC`,
		chainBaseID, chainBaseID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions in chain: %w", err)
	}
	defer rows.Close()

	var versions []models.ConfigVersion
	for rows.Next() {
		var version models.ConfigVersion
		var parentVersionID, chainBase sql.NullInt32
		if err := rows.Scan(&version.ID, &version.DeviceID, &version.VersionHash, &version.StorageType, &version.StoragePath,
			&parentVersionID, &chainBase, &version.ChainPosition, &version.OriginalSize, &version.CompressedSize, &version.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		if parentVersionID.Valid {
			pid := int(parentVersionID.Int32)
			version.ParentVersionID = &pid
		}
		if chainBase.Valid {
			cid := int(chainBase.Int32)
			version.ChainBaseID = &cid
		}
		versions = append(versions, version)
	}

	return versions, nil
}

func (db *DB) GetVersionsForDevice(deviceID int, limit int, offset int) ([]models.ConfigVersion, error) {
	query := `
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE device_id = ? 
		ORDER BY id DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := db.connection.Query(query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []models.ConfigVersion
	for rows.Next() {
		var version models.ConfigVersion
		var parentVersionID, chainBaseID sql.NullInt32
		if err := rows.Scan(&version.ID, &version.DeviceID, &version.VersionHash, &version.StorageType, &version.StoragePath,
			&parentVersionID, &chainBaseID, &version.ChainPosition, &version.OriginalSize, &version.CompressedSize, &version.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		if parentVersionID.Valid {
			pid := int(parentVersionID.Int32)
			version.ParentVersionID = &pid
		}
		if chainBaseID.Valid {
			cid := int(chainBaseID.Int32)
			version.ChainBaseID = &cid
		}
		versions = append(versions, version)
	}

	return versions, nil
}

func (db *DB) SaveVersion(
	deviceID int,
	versionHash string,
	storageType string,
	storagePath string,
	parentVersionID *int,
	chainBaseID *int,
	chainPosition int,
	originalSize uint32,
	compressedSize uint32,
) (*models.ConfigVersion, error) {

	var baseID, parentID interface{}
	if chainBaseID != nil {
		baseID = *chainBaseID
	}
	if parentVersionID != nil {
		parentID = *parentVersionID
	}

	result, err := db.connection.Exec(`
		INSERT INTO config_versions 
		(device_id, version_hash, storage_type, storage_path, parent_version_id, chain_base_id, chain_position, original_size, compressed_size) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deviceID, versionHash, storageType, storagePath, parentID, baseID, chainPosition, originalSize, compressedSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save version: %w", err)
	}

	id, _ := result.LastInsertId()

	version := &models.ConfigVersion{
		ID:              int(id),
		DeviceID:        deviceID,
		VersionHash:     versionHash,
		StorageType:     storageType,
		StoragePath:     storagePath,
		ParentVersionID: parentVersionID,
		ChainBaseID:     chainBaseID,
		ChainPosition:   chainPosition,
		OriginalSize:    originalSize,
		CompressedSize:  compressedSize,
		CreatedAt:       time.Now(),
	}

	log.Printf("saved version %d for device %d: type=%s, chain_pos=%d, path=%s",
		version.ID, deviceID, storageType, chainPosition, storagePath)
	return version, nil
}

func (db *DB) CreateJob(jobType string, status string, payloadJSON *string) (*models.Job, error) {
	result, err := db.connection.Exec(`
		INSERT INTO jobs (type, status, payload_json) 
		VALUES (?, ?, ?)`,
		jobType, status, payloadJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	id, _ := result.LastInsertId()

	job := &models.Job{
		ID:          int(id),
		Type:        jobType,
		Status:      status,
		PayloadJSON: payloadJSON,
		CreatedAt:   time.Now(),
	}

	return job, nil
}

func (db *DB) UpdateJobStatus(jobID int, status string, errorText *string) error {
	_, err := db.connection.Exec(`
		UPDATE jobs SET status = ?, error_text = ?, finished_at = NOW() WHERE id = ?`,
		status, errorText, jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

func (db *DB) GetDiffIndex(leftVersionID, rightVersionID int) (*models.DiffIndex, error) {
	var diffIndex models.DiffIndex
	var storagePath sql.NullString

	err := db.connection.QueryRow(`
		SELECT id, left_version_id, right_version_id, added_lines, removed_lines, diff_storage_path, created_at 
		FROM diff_index 
		WHERE left_version_id = ? AND right_version_id = ?`,
		leftVersionID, rightVersionID,
	).Scan(&diffIndex.ID, &diffIndex.LeftVersionID, &diffIndex.RightVersionID,
		&diffIndex.AddedLines, &diffIndex.RemovedLines, &storagePath, &diffIndex.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get diff index: %w", err)
	}

	if storagePath.Valid {
		diffIndex.DiffStoragePath = &storagePath.String
	}

	return &diffIndex, nil
}

func (db *DB) SaveDiffIndex(leftVersionID, rightVersionID int, addedLines, removedLines int, storagePath *string) (*models.DiffIndex, error) {
	result, err := db.connection.Exec(`
		INSERT INTO diff_index (left_version_id, right_version_id, added_lines, removed_lines, diff_storage_path) 
		VALUES (?, ?, ?, ?, ?)`,
		leftVersionID, rightVersionID, addedLines, removedLines, storagePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save diff index: %w", err)
	}

	id, _ := result.LastInsertId()

	diffIndex := &models.DiffIndex{
		ID:              int(id),
		LeftVersionID:   leftVersionID,
		RightVersionID:  rightVersionID,
		AddedLines:      addedLines,
		RemovedLines:    removedLines,
		DiffStoragePath: storagePath,
		CreatedAt:       time.Now(),
	}

	return diffIndex, nil
}

func (db *DB) Close() error {
	return db.connection.Close()
}

// GetAllFileStates загружает всё содержимое device_file_state одним запросом.
// Возвращает map[hostname]*FileState для быстрого поиска без обращения к БД в цикле.
func (db *DB) GetAllFileStates() (map[string]*FileState, error) {
	rows, err := db.connection.Query(`SELECT hostname, file_size, file_mtime FROM device_file_state`)
	if err != nil {
		return nil, fmt.Errorf("GetAllFileStates: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*FileState)
	for rows.Next() {
		var fs FileState
		var modTime time.Time
		if err := rows.Scan(&fs.Hostname, &fs.Size, &modTime); err != nil {
			return nil, fmt.Errorf("GetAllFileStates scan: %w", err)
		}
		fs.ModTime = truncateToMillis(modTime)
		result[fs.Hostname] = &fs
	}
	return result, nil
}

// ConfigSourceSettings хранит настройки источника конфигов из system_settings.
type ConfigSourceSettings struct {
	Type                string // local, smb, nfs
	Path                string // локальный путь или share path
	SmbServer           string
	SmbShare            string
	SmbUsername         string
	SmbPassword         string
	SmbDomain           string
	NfsServer           string
	NfsPath             string
	ScanIntervalSeconds int
	DiffThreshold       float64
}

// GetConfigSourceSettings читает все настройки источника одним запросом.
func (db *DB) GetConfigSourceSettings() (*ConfigSourceSettings, error) {
	rows, err := db.connection.Query(
		`SELECT settings_key, settings_value FROM system_settings
		 WHERE settings_key IN (
			'config_source_type','config_source_path',
			'smb_username','smb_password','smb_domain',
			'scan_interval_seconds','diff_threshold'
		 )`,
	)
	if err != nil {
		return nil, fmt.Errorf("GetConfigSourceSettings: %w", err)
	}
	defer rows.Close()

	s := &ConfigSourceSettings{
		Type:      "local",
		Path:      "/app/configs",
		SmbDomain: "WORKGROUP",
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case "config_source_type":
			s.Type = value
		case "config_source_path":
			s.Path = value
		case "smb_username":
			s.SmbUsername = value
		case "smb_password":
			s.SmbPassword = value
		case "smb_domain":
			if value != "" {
				s.SmbDomain = value
			}
		case "scan_interval_seconds":
			if n, err := strconv.Atoi(value); err == nil && n >= 5 {
				s.ScanIntervalSeconds = n
			}
		case "diff_threshold":
			if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 {
				s.DiffThreshold = f
			}
		}
	}

	// Парсим путь для SMB: //server/share (также поддерживаем smb://server/share)
	if s.Type == "smb" && s.Path != "" {
		path := strings.TrimSpace(s.Path)
		path = strings.TrimPrefix(path, "smb://")
		path = strings.ReplaceAll(path, "\\", "/")
		path = strings.TrimPrefix(path, "//")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			s.SmbServer = strings.TrimSpace(parts[0])
			s.SmbShare = strings.Trim(strings.TrimSpace(parts[1]), "/")
		}
	}

	// Парсим путь для NFS: //server/path, nfs://server/path или server:/path
	if s.Type == "nfs" && s.Path != "" {
		path := strings.TrimSpace(s.Path)
		path = strings.TrimPrefix(path, "nfs://")
		path = strings.TrimPrefix(path, "//")
		if idx := strings.Index(path, ":/"); idx != -1 {
			// формат server:/path
			s.NfsServer = strings.TrimSpace(path[:idx])
			s.NfsPath = strings.TrimSpace(path[idx+1:])
		} else if parts := strings.SplitN(path, "/", 2); len(parts) == 2 {
			// формат server/path (после снятия //)
			s.NfsServer = strings.TrimSpace(parts[0])
			s.NfsPath = "/" + strings.TrimSpace(parts[1])
		}
	}

	return s, nil
}