Полезные команды:

Удалить остановленные контейнеры
docker rm -f $(docker ps -aq)

Просмотр всех контейнеров
docker ps -a

Запуск отдельных контейнеров
docker-compose -f docker-compose.backend.yml up -d --build
docker-compose -f docker-compose.database.yml up -d --build
docker-compose -f docker-compose.frontend.yml up -d --build
