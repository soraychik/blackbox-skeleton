Приложение будет доступно по адресу `http://localhost:3000`

# Запуск через основной docker-compose.yml
docker-compose up -d --build

# Или отдельно frontend
docker-compose -f docker-compose.frontend.yml up -d --build

Если ошибка "Network Error", проверить:

1. API сервер запущен:
   ```bash
   docker ps | grep api-web
   curl http://localhost:8080/health
   ```

2. Правильный URL API: По умолчанию используется `http://localhost:8080`.
   ```bash
   docker-compose build frontend
   docker-compose up -d frontend
   ```

3. CORS: API сервер должен разрешать запросы с frontend. В текущей конфигурации CORS настроен для всех источников (`*`).

4. Порты: порты 3000 (frontend) и 8080 (API) не заняты другими приложениями.


