# Fitoscout — Мастер-документ архитектуры

> Единый источник правды. Передаётся во все чаты coder.qwen.ai как контекст.

## 1. Контекст проекта

**Fitoscout** — офлайн-first справочник агронома с одним пользователем в двух ролях:
- **Поле (Android)** — чтение справочников + закрытые модули (календарь, реестр, библиотека)
- **ПК (Админка)** — полный CRUD и тяжёлые операции (конвертация, парсинг)
- **NAS DJ220j** — сердце системы (слабое железо, без Docker)

### Ограничения NAS
- CPU: Realtek RTD1296 (ARM64 Cortex-A53, 4 ядра)
- RAM: 512 МБ
- OS: DSM 7.4.1-90080
- Без Docker/Container Manager
- Подключение через Tailscale (mTLS + SSH)

### Роли устройств
- **ПК-админка (web)**: полный CRUD, конвертация, импорт, тяжёлые операции
- **Android-клиент (android)**: чтение + комментарии (кроме замкнутых модулей)

## 2. Стек технологий

| Компонент | Технологии | Версия |
|---|---|---|
| Backend API | Go + MariaDB + REST | Go 1.26.5, MariaDB 10 |
| Frontend Admin | React + TypeScript + Vite + TanStack Query + Tailwind CSS | - |
| Android | Kotlin + Jetpack Compose + Room + WorkManager + Retrofit | - |
| Deploy | Git Bash + scp + ssh + systemd | - |
| Auth | mTLS (client certificates) | TLS 1.2+ |
| Shell | Только Git Bash | PowerShell НЕ используется |

## 3. Архитектурные решения (ADR)

### ADR-001: Clean Architecture
- `domain/` — бизнес-логика, без зависимостей
- `services/` — usecases
- `storage/` — реализации репозиториев
- `api/` — HTTP-хэндлеры (тонкий слой)
- **Dependency Inversion**: api зависит от domain-интерфейсов

### ADR-002: Один бинарник `fitoscoutd`
- API + автоочистка (в горутине)
- Нет отдельного воркера (пока не нужен)
- Если появятся другие фоновые задачи — разделим позже

### ADR-003: Тяжёлые операции в админке на ПК
- Конвертация фото → webp (оригинал + thumbnail + icon)
- Извлечение превью PDF/DJVU
- Извлечение кадра из видео
- Парсинг Markdown
- На сервер передаются **только готовые результаты**
- NAS не выполняет CPU-intensive задачи

### ADR-004: Монотонные версии для LWW (Last-Write-Win)
- `version` — глобальный монотонный счётчик на сервере
- НЕ timestamp (они расходятся на разных устройствах)
- Дельта-синхронизация: `GET /sync?since=N` → `version > N`
- LWW: чья версия выше, та и побеждает при конфликте

### ADR-005: EAV для динамических характеристик
- `attribute_definitions` — метаданные полей (ключ, тип, tooltip, group)
- `attribute_values` — значения в EAV-паттерне
- Гибридное хранение: базовые колонки + EAV
- Пользовательская группировка через `attribute_groups`

### ADR-006: mTLS с ролями через CN сертификата
- Два сертификата: `fitoscout-android-client`, `fitoscout-web-admin`
- Роль определяется по CN (Common Name)
- Проверка через middleware `auth.RoleCheck`
- Дополнительный заголовок `X-Fitoscout-Client` для верификации
- CRL (Certificate Revocation List) в `revoked.txt`

### ADR-007: Префиксы таблиц по модулям
- `plants_`, `diseases_`, `pests_`, `agrochemicals_`...
- Избегаем конфликтов имён
- Упрощает понимание схемы

### ADR-008: Soft-delete + автоочистка
- `deleted_at` — timestamp удаления
- Автоочистка раз в сутки удаляет записи с `deleted_at` старше 30 дней
- Tombstones нужны для синхронизации (чтобы клиент тоже удалил)

### ADR-009: Разделение контента и метаданных
- **Библиотека** (`/volume2/Библиотека/`) — оригиналы книг/видео, "вечное" хранилище
- **Превью книг** (`/volume1/Fitoscout_project/media/covers/`) — метаданные проекта
- **Фото записей** (`/volume1/Fitoscout_project/media/`) — original/thumbnail/icon

### ADR-010: Русскоязычные сообщения
- Пользовательские сообщения (errors, logs) — на русском
- Ключи (Key) — на английском (для grep и машинной обработки)
- Значения (Value) — как есть (методы HTTP, пути, ошибки)

## 4. Структура монорепозитория

```
fitoscout/
├── backend/
│   ├── cmd/
│   │   └── fitoscoutd/
│   │       └── main.go                  # Точка входа
│   ├── internal/
│   │   ├── api/                         # HTTP-хэндлеры
│   │   │   ├── handlers/                # По модулям (plants.go, diseases.go...)
│   │   │   ├── middleware/
│   │   │   ├── router.go
│   │   │   └── server.go
│   │   ├── app/
│   │   │   ├── app.go                   # Инициализация приложения
│   │   │   └── cleanup.go              # Автоочистка
│   │   ├── auth/                        # mTLS + роли
│   │   │   ├── cert.go
│   │   │   ├── middleware.go
│   │   │   ├── restriction.go
│   │   │   └── roles.go
│   │   ├── config/
│   │   │   └── config.go
│   │   ├── domain/                      # 🆕 Доменные модели
│   │   │   ├── entities.go              # Plants, Diseases, EAV...
│   │   │   └── sync.go                 # SyncVersion, LWW
│   │   ├── storage/                     # Реализации репозиториев
│   │   │   ├── mariadb/                 # 🆕 MariaDB
│   │   │   │   └── connection.go
│   │   │   ├── migrations/
│   │   │   ├── config.go
│   │   │   ├── embed.go
│   │   │   ├── migrations.go
│   │   │   ├── repository.go
│   │   │   └── storage.go
│   │   ├── services/                    # 🆕 Бизнес-логика
│   │   ├── errors/
│   │   │   ├── errors.go
│   │   │   └── http.go
│   │   ├── logging/
│   │   │   └── logger.go
│   │   ├── middleware/
│   │   │   ├── logging.go
│   │   │   ├── middleware.go
│   │   │   ├── rate_limit.go
│   │   │   ├── recovery.go
│   │   │   └── request_id.go
│   │   └── tls/
│   │       ├── revocation.go
│   │       └── tls.go
│   ├── migrations/                      # SQL-миграции
│   │   └── 000001_init.up.sql
│   ├── go.mod
│   └── go.sum
├── web/                                 # React-админка (в будущем)
├── android/                             # Android-приложение (в будущем)
├── docs/
│   └── architecture/
│       └── MASTER.md                    # Этот документ
└── deploy/
    ├── build.sh                         # 🆕 Bash вместо PowerShell
    ├── deploy.sh                        # 🆕 Bash вместо PowerShell
    ├── fitoscout.service                # 🆕 Systemd unit
    └── config.example.toml
```

## 5. Схема БД (MariaDB)

### 5.1. Версионируемые таблицы (шаблон)

```sql
-- Базовый шаблон для любой модульной таблицы
CREATE TABLE plants (
    id              VARCHAR(36) PRIMARY KEY,    -- UUIDv7
    plants_name     VARCHAR(255) NOT NULL,      -- префикс модуля
    plants_latin    VARCHAR(255),
    plants_category_id BIGINT,
    -- ... другие базовые поля
    version         BIGINT NOT NULL,            -- монотонная версия для LWW
    created_at      DATETIME(3) NOT NULL,
    updated_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,           -- soft delete
    INDEX idx_plants_version (version),
    INDEX idx_plants_deleted (deleted_at),
    INDEX idx_plants_category (plants_category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### 5.2. Сквозные таблицы

```sql
-- Реестр модулей (динамический)
CREATE TABLE modules (
    id              VARCHAR(32) PRIMARY KEY,     -- 'plants', 'diseases'...
    name            VARCHAR(128) NOT NULL,
    type            ENUM('open','closed') NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      DATETIME(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Категории (иерархия через parent_id)
CREATE TABLE categories (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    parent_id       BIGINT NULL,
    name            VARCHAR(255) NOT NULL,
    icon_path       VARCHAR(512),
    image_path      VARCHAR(512),
    sort_order      INT DEFAULT 0,
    FOREIGN KEY (module_key) REFERENCES modules(id),
    INDEX idx_cat_module (module_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Словари (списки значений для EAV)
CREATE TABLE dictionaries (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    name            VARCHAR(128) NOT NULL,
    description     TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE dictionary_items (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    dictionary_id   BIGINT NOT NULL,
    value           VARCHAR(255) NOT NULL,
    description     TEXT,                          -- tooltip
    terminology_id  BIGINT NULL,                   -- связь с терминологией
    sort_order      INT DEFAULT 0,
    FOREIGN KEY (dictionary_id) REFERENCES dictionaries(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Группы характеристик (пользовательская группировка)
CREATE TABLE attribute_groups (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    name            VARCHAR(128) NOT NULL,
    sort_order      INT DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Определения характеристик
CREATE TABLE attribute_definitions (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    attr_key        VARCHAR(64) NOT NULL,
    data_type       ENUM('int','float','text','date','dict','multi_dict') NOT NULL,
    label           VARCHAR(128) NOT NULL,
    tooltip         TEXT,
    group_id        BIGINT NULL,
    dictionary_id   BIGINT NULL,
    sort_order      INT DEFAULT 0,
    UNIQUE KEY uq_attr_module_key (module_key, attr_key),
    FOREIGN KEY (dictionary_id) REFERENCES dictionaries(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Значения характеристик (EAV)
CREATE TABLE attribute_values (
    entity_module   VARCHAR(32) NOT NULL,
    entity_id       VARCHAR(36) NOT NULL,
    definition_id   BIGINT NOT NULL,
    value_int       BIGINT NULL,
    value_float     DOUBLE NULL,
    value_text      TEXT NULL,
    value_date      DATE NULL,
    value_dict_id   BIGINT NULL,
    version         BIGINT NOT NULL,
    PRIMARY KEY (entity_module, entity_id, definition_id),
    INDEX idx_attr_version (version),
    FOREIGN KEY (definition_id) REFERENCES attribute_definitions(id) ON DELETE CASCADE,
    FOREIGN KEY (value_dict_id) REFERENCES dictionary_items(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Универсальные связи many-to-many
CREATE TABLE entity_links (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    from_module     VARCHAR(32) NOT NULL,
    from_id         VARCHAR(36) NOT NULL,
    to_module       VARCHAR(32) NOT NULL,
    to_id           VARCHAR(36) NOT NULL,
    link_type       VARCHAR(32) NOT NULL,          -- 'related', 'cause', 'treatment'...
    created_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,
    INDEX idx_links_from (from_module, from_id),
    INDEX idx_links_to (to_module, to_id),
    INDEX idx_links_type (link_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Индекс сквозных ссылок [module:id]
CREATE TABLE link_index (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    source_module   VARCHAR(32) NOT NULL,
    source_id       VARCHAR(36) NOT NULL,
    target_module   VARCHAR(32) NOT NULL,
    target_id       VARCHAR(36) NOT NULL,
    INDEX idx_lnk_source (source_module, source_id),
    INDEX idx_lnk_target (target_module, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Фото записей (не для библиотеки!)
CREATE TABLE record_images (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    entity_id       VARCHAR(36) NOT NULL,
    original_path   VARCHAR(512) NOT NULL,         -- /volume1/Fitoscout_project/media/originals/
    thumbnail_path  VARCHAR(512) NOT NULL,         -- /volume1/Fitoscout_project/media/thumbnails/
    icon_path       VARCHAR(512) NOT NULL,         -- /volume1/Fitoscout_project/media/icons/
    is_primary      BOOLEAN DEFAULT FALSE,
    sort_order      INT DEFAULT 0,
    created_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,
    INDEX idx_img_entity (module_key, entity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Состояние синхронизации по устройствам
CREATE TABLE sync_state (
    device_id       VARCHAR(64) PRIMARY KEY,        -- CN из сертификата
    last_version    BIGINT NOT NULL DEFAULT 0,
    last_sync_at    DATETIME(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Комментарии и задачи
CREATE TABLE comments (
    id              VARCHAR(36) PRIMARY KEY,
    module_key      VARCHAR(32) NOT NULL,
    entity_id       VARCHAR(36) NOT NULL,
    comments_type   ENUM('comment','general','task') NOT NULL,
    comments_text   TEXT NOT NULL,
    comments_status ENUM('new','in_progress','done') DEFAULT 'new',
    version         BIGINT NOT NULL,
    created_at      DATETIME(3) NOT NULL,
    updated_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,
    INDEX idx_cmt_entity (module_key, entity_id),
    INDEX idx_cmt_status (comments_status),
    INDEX idx_cmt_version (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Библиотека (книги и видео)
CREATE TABLE library (
    id              VARCHAR(36) PRIMARY KEY,
    library_category_id BIGINT NOT NULL,
    library_title   VARCHAR(255) NOT NULL,
    library_author  VARCHAR(255),
    library_format  ENUM('pdf','djvu','mp4','mkv') NOT NULL,
    library_file_path VARCHAR(512) NOT NULL,       -- /volume2/Библиотека/{категория}/...
    library_cover_path VARCHAR(512),               -- /volume1/Fitoscout_project/media/covers/
    library_size_bytes BIGINT,
    version         BIGINT NOT NULL,
    created_at      DATETIME(3) NOT NULL,
    updated_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,
    INDEX idx_lib_cat (library_category_id),
    INDEX idx_lib_version (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Миграции
CREATE TABLE schema_migrations (
    version         INT PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    applied_at      BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 5.3. Специфичные поля модулей (добавить в соответствующие таблицы)

**Растения (`plants_`):** ~50 характеристик через EAV + базовые:
- `plants_name`, `plants_latin`, `plants_category_id`, `plants_description`

**Болезни (`diseases_`):**
- `diseases_name`, `diseases_parent_id`, `diseases_is_pathogens_group`, `diseases_symptoms`

**Вредители (`pests_`):**
- `pests_name`, `pests_parent_id`, `pests_description`

**Агрохимия (`agrochemicals_`):**
- `agrochemicals_name`, `agrochemicals_manufacturer`, `agrochemicals_form` (SP/WG/EC/SC/...)
- `agrochemicals_is_fertilizer` BOOLEAN
- Таблица `fertilizer_composition` (для удобрений): элемент + доля (%)

**Действующие вещества (`active_substances_`):**
- `active_substances_name`, `active_substances_cas`

**Терминология (`terminologies_`):**
- `terminologies_name`, `terminologies_short_description` (tooltip), `terminologies_full_description`

**Статьи (`articles_`):**
- `articles_name`, `articles_content` (Markdown), `articles_html` (предрендеренный в админке)

**Реестр питомника (`registry_`):**
- `registry_article` VARCHAR UNIQUE, `registry_is_lost` BOOLEAN, `registry_plant_id` (FK → plants)

**Календарь (`calendar_`):**
- `calendar_title`, `calendar_event_at` DATETIME, `calendar_before_event` INT (минут), `calendar_repeat_after` INT (дней)

## 6. API-контракты

### 6.1. Базовые endpoints
```
GET  /api/v1/healthz
GET  /api/v1/auth/whoami
```

### 6.2. Sync API
```
GET  /api/v1/sync?since={version}
→ Дельта: все записи с version > since (с soft-deleted)

POST /api/v1/sync
Body: {changes: [{module, id, data, version}]}
← Применяет изменения клиента (LWW)
→ Возвращает принятые + дельту с сервера
```

### 6.3. CRUD для модулей
```
GET    /api/v1/{module}                    [android, web]
GET    /api/v1/{module}/{id}               [android, web]
POST   /api/v1/{module}                    [web only]
PUT    /api/v1/{module}/{id}               [web only]
DELETE /api/v1/{module}/{id}               [web only]
```

### 6.4. Media upload (только web)
```
POST /api/v1/media/upload
Body: multipart/form-data (уже сконвертированные webp)
  - original (webp)
  - thumbnail (webp)
  - icon (webp)
← Возвращает пути к файлам
```

### 6.5. Library upload (только web)
```
POST /api/v1/library/upload
Body: multipart/form-data
  - file (PDF/DJVU)
  - cover (превью, создано в админке)
  - metadata (JSON: title, author, category)
← Возвращает запись библиотеки
```

### 6.6. Streaming (для чтения книг на Android)
```
GET /api/v1/library/{id}/stream [android, web]
→ HTTP Range requests для PDF/DJVU
```

## 7. Правила именования

| Сущность | Правила | Пример |
|---|---|---|
| Таблицы | префикс модуля + множ. число | `plants`, `categories` |
| Поля | префикс модуля + snake_case | `plants_name`, `diseases_symptoms` |
| API endpoints | множ. число, RESTful | `/api/v1/plants`, `/api/v1/plants/{id}` |
| Переменные Go | camelCase | `plantsName` |
| Константы Go | PascalCase | `RoleWeb`, `CodeNotFound` |
| Пакеты Go | lowercase | `auth`, `storage` |

## 8. Соглашения о коде

### 8.1. Error handling (на русском)
```go
errors.Unauthorized("требуется клиентский сертификат")
errors.Forbidden("недостаточно прав для этой операции")
errors.NotFound("растение не найдено")
errors.Validation("ошибка валидации", details)
errors.Conflict("запись была изменена другим пользователем")
errors.Duplicate("запись с таким именем уже существует")
errors.RateLimited("превышен лимит запросов")
errors.Internal("внутренняя ошибка сервера")
```

### 8.2. Logging (сообщения на русском, ключи на английском)
```go
logger.Info("запрос выполнен",
    logging.Field{Key: "method", Value: "GET"},
    logging.Field{Key: "path", Value: "/api/v1/plants"},
    logging.Field{Key: "status", Value: 200},
    logging.Field{Key: "duration_ms", Value: 45},
)

logger.Error("ошибка подключения к БД",
    logging.Field{Key: "error", Value: err.Error()},
    logging.Field{Key: "dsn", Value: cfg.Database.DSN},
)
```

### 8.3. UUIDv7 для ID
- Использовать `github.com/google/uuid`
- `uuid.Must(uuid.NewV7()).String()`

### 8.4. Timestamps в миллисекундах Unix
- В Go: `time.Now().UnixMilli()`
- В БД: `DATETIME(3)` (MariaDB) или `BIGINT`

### 8.5. Конфигурация
- Формат: TOML (`github.com/pelletier/go-toml/v2`)
- Валидация при загрузке
- Дефолтные значения через `applyDefaults()`

## 9. Модули (11)

| # | Модуль | Префикс | Тип | Особенности |
|---|---|---|---|---|
| 1 | Растения | `plants_` | обычный | ~50 характеристик через EAV |
| 2 | Болезни | `diseases_` | обычный | иерархия род→вид, `is_pathogens_group` |
| 3 | Вредители | `pests_` | обычный | иерархия род→вид |
| 4 | Агрохимия | `agrochemicals_` | обычный | плоский список, удобрения отдельно |
| 5 | Действующие вещества | `active_substances_` | обычный | связь с Агрохимией, баковые смеси |
| 6 | Терминология | `terminologies_` | обычный | глоссарий, тултипы |
| 7 | Статьи | `articles_` | обычный | Markdown + `[module:id]`, `[image:id]` |
| 8 | Реестр питомника | `registry_` | замкнутый | уникальные артикулы, `is_lost` |
| 9 | Календарь | `calendar_` | замкнутый | напоминания, повторения |
| 10 | Библиотека | `library_` | замкнутый | PDF/DJVU/MP4, streaming |
| 11 | Комментарии | `comments_` | замкнутый | типы: comment/general/task |

## 10. Роли и права доступа

| Роль | Сертификат CN | Права |
|---|---|---|
| **ПК-админка** | `fitoscout-web-admin` | Полный CRUD всего |
| **Android-клиент** | `fitoscout-android-client` | Чтение + комментарии (исключения для замкнутых) |

**Замкнутые модули** (полный CRUD с Android): Реестр, Календарь, Библиотека, Комментарии

## 11. Структура на NAS

```
/volume1/Fitoscout_project/
├── bin/
│   ├── fitoscoutd              # Единственный бинарник
│   └── start.sh                # (legacy, использовать systemd)
├── certs/
│   ├── ca.crt                  # Корневой сертификат
│   ├── server.crt              # Сертификат сервера
│   ├── server.key              # Приватный ключ сервера
│   └── revoked.txt             # CRL (sha256 fingerprints)
├── config/
│   └── config.toml             # Конфигурация
├── logs/
│   └── fitoscout.log           # Логи приложения
├── media/
│   ├── originals/              # Оригиналы фото записей (webp)
│   ├── thumbnails/             # Превью 256×256 (webp)
│   ├── icons/                  # Иконки 64×64 (webp)
│   └── covers/                 # 🆕 Превью книг/видео
└── tmp/                        # Временные файлы

/volume1/GitRepos/
└── fitoscout.git/              # Bare Git repository

/volume2/Библиотека/            # Вечное хранилище книг/видео
└── {категория}/
    └── {книга}.pdf             # Только оригиналы, без превью
```

## 12. Systemd service

**Файл:** `/etc/systemd/system/fitoscout.service`

```ini
[Unit]
Description=Fitoscout API Server
After=network.target mariadb.service
Wants=mariadb.service

[Service]
Type=simple
User=root
WorkingDirectory=/volume1/Fitoscout_project
ExecStart=/volume1/Fitoscout_project/bin/fitoscoutd /volume1/Fitoscout_project/config/config.toml
Restart=always
RestartSec=5
StandardOutput=append:/volume1/Fitoscout_project/logs/fitoscout.log
StandardError=append:/volume1/Fitoscout_project/logs/fitoscout.log
TimeoutStopSec=30
KillMode=mixed
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

**Команды:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable fitoscout
sudo systemctl start fitoscout
sudo systemctl status fitoscout
```

## 13. Фичи (5)

| # | Фича | Описание |
|---|---|---|
| 1 | Сквозные ссылки | Формат `[module:id]`, `link_index` для быстрого поиска |
| 2 | Неопознано. Задачи | `comments_type=task`, фото через `media_links` |
| 3 | Баковые смеси | Матрица совместимости + правила SP→WG→EC→SC |
| 4 | Автоочистка | Удаление `deleted_at` старше 30 дней, ежедневно |
| 5 | Стриминг книг | HTTP Range requests для PDF/DJVU |

## 14. Доработки (4)

| # | Доработка | Модуль |
|---|---|---|
| 1 | Химический состав удобрений | Агрохимия |
| 2 | Извлечение превью | Библиотека (в админке на ПК!) |
| 3 | Markdown для статей | Статьи |
| 4 | Группировка характеристик | EAV |