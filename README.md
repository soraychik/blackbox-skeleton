# BlackBox Config Management System

Система для управления конфигурационными файлами с поддержкой удаленных файловых систем и версиониованием.

## 🚀 Быстрый старт

```bash
# Клонировать репозиторий
git clone <repository-url>
cd blackbox-skeleton

# Собрать и запустить
docker-compose up -d --build

# Посмотреть доступные скрипты
./scripts/help.sh
```

Система будет доступна:
- **С nginx (по ТЗ):** http://localhost — единая точка входа (порт 80)
- **Без nginx:** http://localhost:3001 — фронт, http://localhost:8080 — API

## Nginx (reverse proxy по ТЗ)

По ТЗ (п. 1.5–1.6) веб-сервер — **nginx** (reverse proxy, статика). В проекте добавлен сервис `nginx` как единая точка входа:

| Путь       | Назначение                          |
|------------|-------------------------------------|
| `http://localhost/`      | Frontend (React)                    |
| `http://localhost/api/`  | API (прокси на api-web:8080)        |

**Запуск с nginx:** после `docker-compose up -d` открывайте http://localhost

**Чтобы фронт ходил в API через nginx**, в `.env` задайте и пересоберите фронт:
```env
REACT_APP_API_URL=/api
```
Затем: `docker-compose up -d --build`

Конфиг nginx: `nginx/nginx.conf`. Для HTTPS (по п. 1.9 ТЗ) позже можно добавить сертификаты и `listen 443 ssl`.

## Архитектура

Система состоит из следующих компонентов:
- **nginx** — reverse proxy, единая точка входа (порт 80)
- Scheduler - основной сервис для мониторинга конфигурационных файлов
- API Web - REST API для доступа к данным
- Frontend - веб-интерфейс управления
- MySQL - база данных для хранения версий
- MinIO - объектное хранилище для файлов

## 🛠️ Основные команды

```bash
# Сборка всех сервисов
docker-compose build

# Запуск всех сервисов
docker-compose up -d

# Сборка и запуск всех сервисов
docker-compose up -d --build

# Остановка всех сервисов
docker-compose down

# Просмотр логов
docker-compose logs -f scheduler

# Сброс системы (только для разработки)
./scripts/manual-reset.sh

# Проверка статуса системы
./scripts/check-status.sh
```

## Конфигурация

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

### Файловые сервера

#### Основной сервер
- `FILE_SERVER_TYPE` - тип файловой системы (nfs/smb/local)
- `NFS_SERVER` - адрес NFS сервера
- `NFS_SHARE_PATH` - путь к шаре NFS
- `NFS_MOUNT_POINT` - точка монтирования NFS
- `SMB_SERVER` - адрес SMB сервера
- `SMB_SHARE_NAME` - имя шары SMB
- `SMB_MOUNT_POINT` - точка монтирования SMB
- `SMB_USERNAME` - пользователь SMB
- `SMB_PASSWORD` - пароль SMB
- `SMB_DOMAIN` - домен SMB

#### Дополнительные сервера
- `FILE_SERVER_2_ENABLED` - включение второго сервера (true/false)
- `FILE_SERVER_2_TYPE` - тип файловой системы второго сервера
- `FILE_SERVER_2_SERVER` - адрес второго сервера
- `FILE_SERVER_2_SHARE_PATH` - путь к шаре второго сервера
- `FILE_SERVER_2_MOUNT_POINT` - точка монтирования второго сервера

## Масштабирование файловых серверов

Система поддерживает работу с множественными файловыми серверами:

### Добавление нового сервера

1. Добавьте переменные окружения для нового сервера в `.env`:
```env
FILE_SERVER_3_ENABLED=true
FILE_SERVER_3_TYPE=nfs
FILE_SERVER_3_SERVER=192.168.50.151
FILE_SERVER_3_SHARE_PATH=/srv/share
FILE_SERVER_3_MOUNT_POINT=/mnt/nfs3
```

2. Обновите код в `internal/fileserver/manager.go` для обработки конфигурации

### Мониторинг состояния

Система автоматически проверяет состояние всех серверов:
- Health check каждые 60 секунд
- Автоматическое перемонтирование при сбоях
- Параллельная обработка файлов

### Уникальность устройств

Имена устройств формируются как `{serverID}-{fileName}` для обеспечения уникальности:
- `server1-config.json`
- `server2-config.json`
- `local-config.json`

## Структура проекта

```
blackbox-skeleton/
├── api-web/                 # API сервис
├── scheduler/               # Основной планировщик
│   ├── cmd/                # Команды
│   ├── internal/           # Внутренние пакеты
│   │   ├── database/       # Работа с БД
│   │   ├── fileprocessor/  # Обработка файлов
│   │   ├── fileserver/     # Управление файловыми серверами
│   │   ├── models/         # Модели данных
│   │   ├── remotefs/       # Удаленные файловые системы
│   │   └── storage/        # Хранилище
│   └── configs/            # Конфигурационные файлы
├── frontend/               # Веб-интерфейс
├── scripts/               # Скрипты
└── docs/                  # Документация
```

## Сборка

### Разработка

```bash
# Сборка scheduler
cd scheduler
go build -o scheduler .

# Сборка API web
cd api-web
go build -o api-web .

# Сборка frontend
cd frontend
npm install
npm run build
```

### Продакшн

```bash
# Сборка всех сервисов
docker-compose build

# Запуск в продакшн режиме
docker-compose up -d
```

## Мониторинг

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

## Сброс системы (только для разработки)

⚠️ Внимание: Функция сброса предназначена только для разработки!

### Использование скрипта сброса

1. Включите сброс в `.env` файле:
```env
DEV_FULL_RESET=true
```

2. Запустите скрипт:
```bash
./scripts/manual-reset.sh
```

3. Подтвердите сброс (дважды: через ENV и вручную)

### Что делает скрипт

- Останавливает все контейнеры
- Удаляет все Docker volumes (БД + MinIO)
- Перезапускает систему с чистыми данными
- Проверяет статус после сброса

### Безопасность

- Двойная защита: ENV переменная + ручное подтверждение
- Отключено по умолчанию для предотвращения случайного сброса
- Только для разработки, не используйте в продакшн

### Проверка состояния

```bash
# Проверить что система чистая после сброса
./scripts/check-status.sh
```

## Траблшутинг

### Проблемы с монтированием NFS

1. Проверьте доступность сервера:
```bash
ping $NFS_SERVER
```

2. Проверьте экспортированные директории:
```bash
showmount -e $NFS_SERVER
```

3. Проверьте права доступа на сервере

### Проблемы с SMB

1. Проверьте доступность сервера:
```bash
ping $SMB_SERVER
```

2. Проверьте доступ к шаре:
```bash
smbclient //$SMB_SERVER/$SMB_SHARE_NAME -U $SMB_USERNAME
```

3. Проверьте учетные данные

### Проблемы с базой данных

1. Проверьте подключение к MySQL:
```bash
docker-compose exec mysql-db mysql -u appuser -p blackbox
```

2. Полный сброс для разработки:
```bash
# Сначала включите в .env: DEV_FULL_RESET=true
# Затем запустите
./scripts/manual-reset.sh
```

3. Проверьте состояние после сброса:
```bash
./scripts/check-status.sh
```

### Проблемы с контейнерами

```bash
# Полная пересборка
docker-compose down -v
docker-compose up -d --build

# Очистка Docker
docker system prune -a
```

## Разработка

### Добавление новых типов файловых систем

1. Реализуйте интерфейс в `internal/remotefs/`
2. Обновите `internal/fileserver/manager.go`
3. Добавьте переменные окружения
4. Обновите документацию

### Добавление новых эндпоинтов API

1. Обновите `api-web/internal/endpoints/`
2. Добавьте модели данных
3. Обновите фронтенд
4. Добавьте тесты

## Безопасность

- Используйте сложные пароли для БД
- Настройте firewall для файловых серверов
- Используйте HTTPS для API в продакшн
- Регулярно обновляйте зависимости

## Лицензия

MIT License