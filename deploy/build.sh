#!/bin/bash
# Сборка fitoscoutd для NAS (linux/arm64). Только Git Bash, без PowerShell.
set -euo pipefail

VERSION="${VERSION:-1.0.0}"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "=== Сборка fitoscoutd для linux/arm64 ==="
echo "Версия: $VERSION"
echo "Коммит: $COMMIT"
echo "Дата сборки: $BUILD_DATE"

if ! command -v go >/dev/null 2>&1; then
    echo "✗ Go не найден в PATH. Установите Go 1.26+ и повторите."
    exit 1
fi

LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.buildDate=$BUILD_DATE"

cd "$(dirname "$0")/../backend"

GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
    -ldflags "$LDFLAGS" \
    -o ../deploy/fitoscoutd \
    ./cmd/fitoscoutd

echo "✓ Сборка успешна: deploy/fitoscoutd"