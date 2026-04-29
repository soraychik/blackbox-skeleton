CREATE TABLE IF NOT EXISTS audit_log (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    username     VARCHAR(255) NOT NULL,
    role         VARCHAR(50)  NOT NULL,
    action       VARCHAR(100) NOT NULL,
    details      JSON,
    ip_address   VARCHAR(45)  NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_audit_log_created_at ON audit_log (created_at DESC);
CREATE INDEX idx_audit_log_username   ON audit_log (username);
CREATE INDEX idx_audit_log_action     ON audit_log (action);
