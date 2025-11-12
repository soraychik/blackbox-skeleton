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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    UNIQUE KEY unique_device_version (device_id, version_date)
);