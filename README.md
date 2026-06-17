### Перед началом работы

Установить nginx
```bash
sudo apt update
sudo apt install nginx -y
```
Скопировать готовую статику
```bash
sudo mkdir -p /var/www/blackbox
sudo cp -r frontend/dist/* /var/www/blackbox/
sudo chown -R www-data:www-data /var/www/blackbox
```
Если директория frontend/dist/ отсутсвует
```bash
cd /blackbox/frontend/
npm run build
```
После этого копировать статику!

Далее скопировать и применить конфиг nginx
```bash
sudo cp nginx/nginx.conf /etc/nginx/nginx.conf
sudo nginx -t
sudo systemctl reload nginx
```




### Основные параметры

Настройте параметры в файле `.env` - он уже содержит рабочую конфигурацию.

### База данных

- `DATABASE_HOST` - хост MySQL
- `DATABASE_PORT` - порт MySQL
- `DATABASE_NAME` - имя базы данных
- `DATABASE_USER` - пользователь БД
- `DATABASE_PASSWORD` - пароль пользователя
- `RESET_DATABASE_ON_START` - очистка БД при запуске (true/false)

### Хранилище файлов

- `USE_MINIO` - использование MinIO (true/false)
- `MINIO_ENDPOINT` - endpoint MinIO
- `MINIO_ACCESS_KEY` - ключ доступа MinIO
- `MINIO_SECRET_KEY` - секретный ключ MinIO
- `MINIO_BUCKET_NAME` - имя бакета

### Продакшн

```bash
# Сборка всех сервисов
docker-compose build

# Запуск в продакшн режиме
docker-compose up -d
```

### Логи

```bash
# Просмотр логов scheduler
docker-compose logs -f scheduler

# Просмотр логов всех сервисов
docker-compose logs -f

# Просмотр логов за последние 100 строк
docker-compose logs --tail=100 scheduler
```

### Статус

```bash
# Статус всех контейнеров
docker-compose ps

# Проверка здоровья сервисов
docker-compose exec scheduler ps aux
```

### Проблемы с базой данных

1. Проверьте подключение к MySQL:
```bash
docker-compose exec mysql-db mysql -u appuser -p blackbox
```

### Проблемы с контейнерами

```bash
# Полная пересборка
docker-compose down -v
docker-compose up -d --build

# Очистка Docker
docker system prune -a
```
