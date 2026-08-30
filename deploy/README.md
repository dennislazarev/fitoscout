# Fitoscout · Деплой (задача #1)

Все команды выполняются в **Git Bash** (PowerShell не используется — ADR проекта).

## 1. Подготовка NAS (однократно)

### 1.1. База данных MariaDB

```bash
ssh -p 7529 tsvl-8@nas
sudo mysql -u root -p
```

```sql
CREATE DATABASE fitoscout CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'fitoscout'@'localhost' IDENTIFIED BY 'СЛОЖНЫЙ_ПАРОЛЬ';
GRANT ALL PRIVILEGES ON fitoscout.* TO 'fitoscout'@'localhost';
FLUSH PRIVILEGES;
```

Миграции применяются **автоматически при старте** fitoscoutd (встроены в бинарник).
Ручной накат (если нужно заранее):

```bash
mysql -u fitoscout -p fitoscout < backend/migrations/000001_init.up.sql
mysql -u fitoscout -p fitoscout -e "SELECT version, name FROM schema_migrations;"
```

### 1.2. Systemd-юнит

```bash
scp -P 7529 deploy/fitoscout.service tsvl-8@nas:/tmp/fitoscout.service
ssh -p 7529 tsvl-8@nas "sudo cp /tmp/fitoscout.service /etc/systemd/system/fitoscout.service"
ssh -p 7529 tsvl-8@nas "sudo systemctl daemon-reload && sudo systemctl enable fitoscout"
```

### 1.3. Конфигурация

```bash
cp deploy/config.example.toml /tmp/config.toml   # отредактируйте пароль БД
scp -P 7529 /tmp/config.toml tsvl-8@nas:/volume1/Fitoscout_project/config/config.toml
```

## 2. Сборка и деплой

```bash
# Сборка под linux/arm64 (VERSION - опционально)
./deploy/build.sh
VERSION=1.1.0 ./deploy/build.sh

# Деплой: stop → резервная копия → scp → start
NAS_USER=tsvl-8 NAS_HOST=nas NAS_PORT=7529 ./deploy/deploy.sh
```

Значения по умолчанию: `NAS_USER=tsvl-8`, `NAS_HOST=nas`, `NAS_PORT=7529`.

## 3. Эксплуатация

```bash
ssh -p 7529 tsvl-8@nas "sudo systemctl status fitoscout"
ssh -p 7529 tsvl-8@nas "sudo systemctl restart fitoscout"

# Логи (json, сообщения на русском)
ssh -p 7529 tsvl-8@nas "tail -f /volume1/Fitoscout_project/logs/fitoscout.log"
```

## 4. Откат

При каждом деплое предыдущий бинарник сохраняется как `fitoscoutd.prev`:

```bash
ssh -p 7529 tsvl-8@nas "
  sudo systemctl stop fitoscout &&
  cp /volume1/Fitoscout_project/bin/fitoscoutd.prev /volume1/Fitoscout_project/bin/fitoscoutd &&
  sudo systemctl start fitoscout
"
```

## 5. Проверка после деплоя

```bash
# С клиентским сертификатом админки (mTLS)
curl --cert web-admin.crt --key web-admin.key --cacert ca.crt \
     https://nas:8443/api/v1/healthz

curl --cert web-admin.crt --key web-admin.key --cacert ca.crt \
     https://nas:8443/api/v1/auth/whoami
```

Ожидаемый ответ `whoami`: `"role": "web"`, `"cn": "fitoscout-web-admin"`.

## 6. Автоочистка

Фоновая задача в fitoscoutd (ADR-008): раз в 24 часа физически удаляет
записи с `deleted_at` старше `retention_days` (по умолчанию 30) и
чистит сиротские `attribute_values` / `link_index`.
Параметры — в секции `[cleanup]` конфига.