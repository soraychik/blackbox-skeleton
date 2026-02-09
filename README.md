```bash
# Сборка всех сервисов одной командой
docker-compose build

# Запуск всех сервисов
docker-compose up -d

# Сборка и запуск всех сервисов
docker-compose up -d --build

# Остановка всех сервисов
docker-compose down

# Удалить остановленные контейнеры
docker rm -f $(docker ps -aq)

# Просмотр всех контейнеров
docker ps -a

# Запуск отдельных контейнеров (если нужно)
docker-compose -f docker-compose.backend.yml up -d --build
docker-compose -f docker-compose.database.yml up -d --build

# Убить процесс по PID
netstat -ao | findstr :80
taskkill -PID [PID] -F
```