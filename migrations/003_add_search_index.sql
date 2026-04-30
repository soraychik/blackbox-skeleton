-- Кэш результатов поиска по паттерну для ускорения повторных запросов.
CREATE TABLE IF NOT EXISTS search_index (
    version_id    INT NOT NULL,
    pattern_hash  CHAR(32) NOT NULL,
    match_count   INT NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (version_id, pattern_hash),
    FOREIGN KEY (version_id) REFERENCES config_versions(id) ON DELETE CASCADE
) ENGINE=InnoDB;