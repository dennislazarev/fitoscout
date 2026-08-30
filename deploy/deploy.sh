#!/bin/bash
# Деплой fitoscoutd на NAS DJ220j через Tailscale (ssh/scp). Только Git Bash.
set -euo pipefail

NAS_USER="${NAS_USER:-tsvl-8}"
NAS_HOST="${NAS_HOST:-nas}"
NAS_PORT="${NAS_PORT:-7529}"
# Имя systemd-сервиса совпадает с deploy/fitoscout.service
SERVICE="fitoscout"
PROJECT_PATH="/volume1/Fitoscout_project"
BINARY_PATH="$(dirname "$0")/fitoscoutd"

if [ ! -f "$BINARY_PATH" ]; then
    echo "✗ Бинарник не найден: $BINARY_PATH"
    echo "  Сначала запусти build.sh"
    exit 1
fi

echo "=== Деплой fitoscoutd на NAS ==="

echo "→ Остановка сервиса..."
ssh -p "$NAS_PORT" "$NAS_USER@$NAS_HOST" \
    "sudo systemctl stop $SERVICE 2>/dev/null || true"

echo "→ Резервная копия текущего бинарника..."
ssh -p "$NAS_PORT" "$NAS_USER@$NAS_HOST" \
    "cp $PROJECT_PATH/bin/fitoscoutd $PROJECT_PATH/bin/fitoscoutd.prev 2>/dev/null || true"

echo "→ Загрузка нового бинарника..."
scp -P "$NAS_PORT" "$BINARY_PATH" "$NAS_USER@$NAS_HOST:$PROJECT_PATH/bin/fitoscoutd"

echo "→ Установка прав..."
ssh -p "$NAS_PORT" "$NAS_USER@$NAS_HOST" \
    "chmod 755 $PROJECT_PATH/bin/fitoscoutd"

echo "→ Запуск сервиса..."
ssh -p "$NAS_PORT" "$NAS_USER@$NAS_HOST" \
    "sudo systemctl start $SERVICE"

echo "→ Проверка статуса..."
ssh -p "$NAS_PORT" "$NAS_USER@$NAS_HOST" \
    "sudo systemctl --no-pager status $SERVICE | head -n 5 || true"

echo "✓ Деплой завершён"