-- Кэш состояния файлов на диске для mtime/size pre-check.
-- Позволяет пропускать файлы которые не изменились без чтения контента.
CREATE TABLE IF NOT EXISTS device_file_state (
    hostname    VARCHAR(255) NOT NULL,
    file_size   BIGINT       NOT NULL,
    file_mtime  DATETIME(3)  NOT NULL,
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (hostname)
) ENGINE=InnoDB;