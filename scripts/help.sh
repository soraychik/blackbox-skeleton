#!/bin/bash
# Список всех доступных скриптов

echo "=== Доступные скрипты BlackBox ==="
echo ""

echo "🔄 Скрипты сброса:"
echo "  ./scripts/manual-reset.sh    - Полный сброс системы разработки (требует DEV_FULL_RESET=true)"
echo "  ./scripts/check-status.sh    - Проверка статуса системы и данных"
echo ""

echo "Примеры использования:"
echo "  # Включить сброс в разработке"
echo "  echo 'DEV_FULL_RESET=true' >> .env"
echo "  ./scripts/manual-reset.sh"
echo ""
echo "  # Проверить систему после сброса"
echo "  ./scripts/check-status.sh"
echo ""

echo "📖 Документация:"
echo "  docs/DEV_RESET_GUIDE.md      - Подробное руководство по сбросу"
echo "  docs/MULTI_SERVER_ARCHITECTURE.md - Архитектура серверов"