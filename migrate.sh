#!/bin/sh
set -e

HOST="${DATABASE_HOST:-mysql-db}"
PORT="${DATABASE_PORT:-3306}"
USER="${DATABASE_USER:-appuser}"
PASS="${DATABASE_PASSWORD:-apppassword}"
DB="${DATABASE_NAME:-blackbox}"
MIGRATIONS_DIR="/migrations"

MYSQL="mysql -h${HOST} -P${PORT} -u${USER} -p${PASS} ${DB}"

echo "=== BlackBox DB Migrations ==="

# Создаём таблицу для отслеживания применённых миграций если её нет
$MYSQL <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    VARCHAR(255) NOT NULL PRIMARY KEY,
    applied_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;
SQL

# Применяем все файлы из /migrations в алфавитном порядке
for file in $(ls ${MIGRATIONS_DIR}/*.sql 2>/dev/null | sort); do
    version=$(basename "$file")

    # Проверяем — уже применена?
    already=$($MYSQL -sN -e "SELECT COUNT(*) FROM schema_migrations WHERE version='${version}'")
    if [ "$already" -gt "0" ]; then
        echo "  skip: ${version} (already applied)"
        continue
    fi

    echo "  apply: ${version}"
    $MYSQL < "$file"

    # Записываем что применили
    $MYSQL -e "INSERT INTO schema_migrations (version) VALUES ('${version}')"
    echo "  done: ${version}"
done

echo "=== Migrations complete ==="