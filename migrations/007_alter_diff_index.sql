-- migrations/007_alter_diff_index.sql

-- Добавляем diff_storage_path если её нет
SET @col_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
    WHERE TABLE_SCHEMA = DATABASE() 
      AND TABLE_NAME = 'diff_index' 
      AND COLUMN_NAME = 'diff_storage_path'
);
SET @sql = IF(@col_exists = 0,
    'ALTER TABLE diff_index ADD COLUMN diff_storage_path VARCHAR(512) NULL',
    'SELECT ''diff_storage_path already exists'' AS info'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Удаляем diff_content если она есть
SET @col_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
    WHERE TABLE_SCHEMA = DATABASE() 
      AND TABLE_NAME = 'diff_index' 
      AND COLUMN_NAME = 'diff_content'
);
SET @sql = IF(@col_exists = 1,
    'ALTER TABLE diff_index DROP COLUMN diff_content',
    'SELECT ''diff_content does not exist'' AS info'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;