# Development Reset Guide

## Overview

Система имеет защищенный механизм сброса данных для разработки, который предотвращает случайное удаление данных в продакшн.

## Safety Mechanisms

### Double Protection
1. **Environment Variable**: `DEV_FULL_RESET=true` требуется для активации
2. **Manual Confirmation**: Интерактивное подтверждение при запуске

### Default State
- `DEV_FULL_RESET=false` по умолчанию
- Безопасно для продакшн окружений
- Предотвращает автоматические сбросы

## Usage

### Step 1: Enable Reset
В файле `.env`:
```env
DEV_FULL_RESET=true
```

### Step 2: Run Reset Script
```bash
./scripts/manual-reset.sh
```

### Step 3: Confirm Reset
Скрипт запросит подтверждение:
```
Are you sure you want to reset ALL development data? (y/N):
```

## What Gets Reset

1. **MySQL Database**: Все таблицы удаляются и создаются заново
2. **MinIO Storage**: Все объекты в бакете удаляются
3. **Docker Volumes**: Все volumes удаляются
4. **Local Cache**: Временные файлы очищаются

## Post-Reset Verification

После сброса проверьте систему:
```bash
./scripts/check-status.sh
```

Ожидаемый результат:
- 0 устройств в БД
- 0 версий в БД  
- Пустой MinIO бакет
- Чистый локальный кеш

## Development Workflow

### Typical Reset Cycle
```bash
# 1. Enable reset
echo "DEV_FULL_RESET=true" >> .env

# 2. Reset system
./scripts/manual-reset.sh

# 3. Test with clean data
# (run tests, experiments, etc.)

# 4. Disable reset for safety
sed -i 's/DEV_FULL_RESET=true/DEV_FULL_RESET=false/' .env
```

### Git Safety
Для разработки можно временно игнорировать изменения в .env файле:
```bash
# Игнорировать изменения в .env файле
git update-index --assume-unchanged .env
```

## Troubleshooting

### Reset Fails
```bash
# Проверить состояние контейнеров
docker-compose ps

# Проверить volumes
docker volume ls

# Ручная очистка при необходимости
docker-compose down -v
docker system prune -f
```

### Permissions Error
```bash
# Проверить права на скрипт
chmod +x ./scripts/manual-reset.sh

# Проверить права на docker
sudo usermod -aG docker $USER
```

### Services Not Starting
```bash
# Проверить логи
docker-compose logs -f

# Пересобрать образы
docker-compose build --no-cache

# Проверить доступность портов
netstat -tulpn | grep -E ':(3000|8080|3306|9000)'
```

## Best Practices

### Before Reset
1. Сделайте бэкап важных данных
2. Запишите текущее состояние
3. Убедитесь что это dev окружение

### After Reset
1. Проверьте что всё работает
2. Отключите сброс обратно
3. Сделайте commit изменений

### Team Collaboration
- Используйте разные `.env` файлы для dev/prod
- Документируйте процесс для команды
- Установите clear naming conventions

## Example Configurations

### Development .env
```env
# Development settings
DEV_FULL_RESET=true
DEBUG=true
LOG_LEVEL=debug

# Database settings (can be dev-specific)
DATABASE_HOST=mysql-db
DATABASE_NAME=blackbox_dev
```

### Production .env
```env
# Production settings
DEV_FULL_RESET=false
DEBUG=false
LOG_LEVEL=error

# Production database
DATABASE_HOST=prod-db-server
DATABASE_NAME=blackbox_prod
```

## Security Notes

- Никогда не используйте `DEV_FULL_RESET=true` в продакшн
- Используйте разные переменные для разных окружений
- Регулярно проверяйте git status на случайные изменения

## Migration from Old Reset

Если вы раньше использовали `RESET_DATABASE_ON_START`:

1. Замените на `DEV_FULL_RESET=true`
2. Используйте `./scripts/manual-reset.sh` вместо автоматического сброса
3. Обновите CI/CD скрипты если нужно