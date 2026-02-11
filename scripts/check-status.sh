#!/bin/bash
# Проверка статуса системы и подтверждение работы сброса

echo "=== Проверка статуса системы BlackBox ==="
echo ""

# Проверка запущенных контейнеров
echo "=== Статус контейнеров ==="
docker-compose ps
echo ""

# Проверка содержимого базы данных
echo "=== Содержимое базы данных ==="
if docker-compose ps mysql-db | grep -q "Up"; then
    echo "Таблицы в базе данных:"
    docker-compose exec -T mysql-db mysql -u appuser -papppassword blackbox -e "SHOW TABLES;" 2>/dev/null || echo "Подключение к базе данных не удалось"
    
    echo ""
    echo "Количество устройств:"
    docker-compose exec -T mysql-db mysql -u appuser -papppassword blackbox -e "SELECT COUNT(*) as device_count FROM devices;" 2>/dev/null || echo "Таблица devices отсутствует или подключение не удалось"
    
    echo ""
    echo "Количество версий:"
    docker-compose exec -T mysql-db mysql -u appuser -papppassword blackbox -e "SELECT COUNT(*) as version_count FROM config_versions;" 2>/dev/null || echo "Таблица config_versions отсутствует или подключение не удалось"
else
    echo "Контейнер MySQL не запущен"
fi
echo ""

# Проверка содержимого MinIO
echo "=== Содержимое MinIO ==="
if docker-compose ps minio | grep -q "Up"; then
    echo "Проверка содержимого бакета MinIO..."
    docker-compose exec -T minio sh -c "
        if command -v mc &> /dev/null; then
            mc alias set local http://localhost:9000 \$MINIO_ROOT_USER \$MINIO_ROOT_PASSWORD 2>/dev/null
            if mc ls local/blackbox-configs/ 2>/dev/null; then
                echo 'Объектов в бакете:'
                mc ls local/blackbox-configs/ | wc -l | sed 's/^/  /'
            else
                echo '  Бакет пуст или не существует'
            fi
        else
            echo '  Клиент MinIO недоступен'
        fi
    "
else
    echo "Контейнер MinIO не запущен"
fi
echo ""

# Проверка локального кеша
echo "=== Локальный кеш ==="
if [ -d "./scheduler/archived_configs" ]; then
    cache_size=$(du -sh ./scheduler/archived_configs 2>/dev/null | cut -f1)
    echo "Размер локального кеша: $cache_size"
    
    cache_files=$(find ./scheduler/archived_configs -type f 2>/dev/null | wc -l)
    echo "Файлов в локальном кеше: $cache_files"
else
    echo "Директория локального кеша не найдена"
fi
echo ""

# Итог
echo "=== Итог ==="
echo "Если вы только что выполнили сброс:"
echo "- База данных должна содержать 0 устройств и 0 версий"
echo "- MinIO должен быть пустым"
echo "- Локальный кеш должен быть пустым или минимальным"
echo ""
echo "Посетите http://localhost:3000 для просмотра статуса веб-интерфейса."