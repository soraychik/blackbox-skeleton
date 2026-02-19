CREATE DATABASE IF NOT EXISTS blackbox CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE blackbox;

DROP TABLE IF EXISTS audit;
DROP TABLE IF EXISTS diff_index;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS config_versions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS devices;

CREATE TABLE devices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL UNIQUE,
    mgmt_ip VARCHAR(45),
    vendor VARCHAR(64),
    model VARCHAR(128),
    tags JSON,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE config_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    device_id INT NOT NULL,
    version_hash CHAR(64) NOT NULL,
    storage_type ENUM('base','diff') NOT NULL,
    storage_path VARCHAR(512) NOT NULL,
    parent_version_id INT NULL,
    chain_base_id INT NULL,
    chain_position INT NOT NULL DEFAULT 0,
    original_size INT UNSIGNED,
    compressed_size INT UNSIGNED,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_version_id) REFERENCES config_versions(id) ON DELETE SET NULL,
    FOREIGN KEY (chain_base_id) REFERENCES config_versions(id) ON DELETE SET NULL,
    INDEX idx_device_created (device_id, created_at),
    INDEX idx_hash (version_hash),
    INDEX idx_chain_base (chain_base_id)
) ENGINE=InnoDB;

CREATE TABLE diff_index (
    id INT AUTO_INCREMENT PRIMARY KEY,
    left_version_id INT NOT NULL,
    right_version_id INT NOT NULL,
    added_lines INT NOT NULL DEFAULT 0,
    removed_lines INT NOT NULL DEFAULT 0,
    diff_content TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (left_version_id) REFERENCES config_versions(id) ON DELETE CASCADE,
    FOREIGN KEY (right_version_id) REFERENCES config_versions(id) ON DELETE CASCADE,
    UNIQUE KEY uq_pair (left_version_id, right_version_id)
) ENGINE=InnoDB;

CREATE TABLE jobs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    status ENUM('pending','running','done','failed') NOT NULL DEFAULT 'pending',
    payload_json JSON,
    started_at DATETIME,
    finished_at DATETIME,
    error_text TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    login VARCHAR(128) NOT NULL UNIQUE,
    pass_hash VARCHAR(256) NOT NULL,
    role ENUM('admin','engineer','operator') NOT NULL DEFAULT 'operator',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME
) ENGINE=InnoDB;

CREATE TABLE audit (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NULL,
    action VARCHAR(128) NOT NULL,
    target_type VARCHAR(64),
    target_id VARCHAR(64),
    payload_json JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_audit_created (created_at),
    INDEX idx_audit_user (user_id)
) ENGINE=InnoDB;
