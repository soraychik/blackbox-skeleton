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

# Ждём пока MySQL реально начнёт принимать подключения
echo "Waiting for MySQL to accept connections..."
for i in $(seq 1 30); do
    if $MYSQL -e "SELECT 1" >/dev/null 2>&1; then
        echo "MySQL is ready"
        break
    fi
    echo "  attempt $i/30: not ready, retrying in 2s..."
    sleep 2
done

# Финальная проверка
if ! $MYSQL -e "SELECT 1" >/dev/null 2>&1; then
    echo "ERROR: MySQL not reachable after 60 seconds"
    exit 1
fi

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

    already=$($MYSQL -sN -e "SELECT COUNT(*) FROM schema_migrations WHERE version='${version}'")
    if [ "$already" -gt "0" ]; then
        echo "  skip: ${version} (already applied)"
        continue
    fi

    echo "  apply: ${version}"
    $MYSQL < "$file"

    $MYSQL -e "INSERT INTO schema_migrations (version) VALUES ('${version}')"
    echo "  done: ${version}"
done

echo "=== Migrations complete ==="