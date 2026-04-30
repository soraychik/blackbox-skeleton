-- Таблица журнала действий пользователей.
CREATE TABLE IF NOT EXISTS audit_log (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    username     VARCHAR(255) NOT NULL,
    role         VARCHAR(50)  NOT NULL,
    action       VARCHAR(100) NOT NULL,
    details      JSON,
    ip_address   VARCHAR(45)  NOT NULL DEFAULT '',
    INDEX idx_audit_log_created_at (created_at DESC),
    INDEX idx_audit_log_username   (username),
    INDEX idx_audit_log_action     (action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;