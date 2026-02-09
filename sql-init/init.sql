USE blackbox;

CREATE TABLE IF NOT EXISTS devices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS config_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    device_id INT NOT NULL,
    version_date DATETIME NOT NULL,
    file_path VARCHAR(512) NOT NULL,
    file_hash VARCHAR(64),
    storage_type ENUM('full', 'diff', 'base') DEFAULT 'full',
    parent_version_id INT NULL,
    minio_object_name VARCHAR(512),
    diff_size_bytes INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_version_id) REFERENCES config_versions(id) ON DELETE SET NULL,
    UNIQUE KEY unique_device_version (device_id, version_date),
    INDEX idx_device_version (device_id, version_date DESC),
    INDEX idx_storage_type (storage_type),
    INDEX idx_parent_version (parent_version_id)
);

CREATE TABLE IF NOT EXISTS storage_snapshots (
    id INT AUTO_INCREMENT PRIMARY KEY,
    device_id INT NOT NULL,
    version_id INT NOT NULL,
    snapshot_type ENUM('full', 'base') DEFAULT 'full',
    minio_object_name VARCHAR(512) NOT NULL,
    file_size_bytes INT NOT NULL,
    compressed_size_bytes INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES config_versions(id) ON DELETE CASCADE,
    INDEX idx_device_snapshot (device_id, snapshot_type),
    INDEX idx_version_snapshot (version_id)
);