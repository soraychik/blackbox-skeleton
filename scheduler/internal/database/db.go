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
