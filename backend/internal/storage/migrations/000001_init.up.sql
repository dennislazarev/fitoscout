-- ============================================================
-- Fitoscout · Миграция 000001 · Исходная схема MariaDB
-- Задача #1: миграция с SQLite на MariaDB (MASTER.md, раздел 5)
--
-- КОПИЯ ДЛЯ РУЧНОГО ПРИМЕНЕНИЯ (mysql-клиентом):
--   mysql -u fitoscout -p fitoscout < migrations/000001_init.up.sql
--
-- Авторитетная копия встроена в бинарник и применяется автоматически:
--   internal/storage/migrations/000001_init.up.sql
-- Держите файлы идентичными.
-- ============================================================

-- Реестр модулей (динамический)
CREATE TABLE IF NOT EXISTS modules (
    id              VARCHAR(32) PRIMARY KEY COMMENT 'plants, diseases, ...',
    name            VARCHAR(128) NOT NULL,
    type            ENUM('open','closed') NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      DATETIME(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Категории (иерархия через parent_id)
CREATE TABLE IF NOT EXISTS categories (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    parent_id       BIGINT NULL,
    name            VARCHAR(255) NOT NULL,
    icon_path       VARCHAR(512),
    image_path      VARCHAR(512),
    sort_order      INT DEFAULT 0,
    INDEX idx_cat_module (module_key),
    CONSTRAINT fk_cat_module FOREIGN KEY (module_key) REFERENCES modules(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Словари (списки значений для EAV)
CREATE TABLE IF NOT EXISTS dictionaries (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    name            VARCHAR(128) NOT NULL,
    description     TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS dictionary_items (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    dictionary_id   BIGINT NOT NULL,
    value           VARCHAR(255) NOT NULL,
    description     TEXT COMMENT 'tooltip',
    terminology_id  BIGINT NULL COMMENT 'связь с терминологией',
    sort_order      INT DEFAULT 0,
    INDEX idx_dict_item_dict (dictionary_id),
    CONSTRAINT fk_dict_item_dict FOREIGN KEY (dictionary_id)
        REFERENCES dictionaries(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Группы характеристик (пользовательская группировка, доработка 4)
CREATE TABLE IF NOT EXISTS attribute_groups (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    name            VARCHAR(128) NOT NULL,
    sort_order      INT DEFAULT 0,
    INDEX idx_attr_group_module (module_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Определения характеристик (EAV, ADR-005)
CREATE TABLE IF NOT EXISTS attribute_definitions (
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
    CONSTRAINT fk_attr_dict FOREIGN KEY (dictionary_id) REFERENCES dictionaries(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Значения характеристик (EAV)
CREATE TABLE IF NOT EXISTS attribute_values (
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
    CONSTRAINT fk_attrval_def FOREIGN KEY (definition_id)
        REFERENCES attribute_definitions(id) ON DELETE CASCADE,
    CONSTRAINT fk_attrval_dict FOREIGN KEY (value_dict_id)
        REFERENCES dictionary_items(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Универсальные связи many-to-many
CREATE TABLE IF NOT EXISTS entity_links (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    from_module     VARCHAR(32) NOT NULL,
    from_id         VARCHAR(36) NOT NULL,
    to_module       VARCHAR(32) NOT NULL,
    to_id           VARCHAR(36) NOT NULL,
    link_type       VARCHAR(32) NOT NULL COMMENT 'related, cause, treatment...',
    created_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,
    INDEX idx_links_from (from_module, from_id),
    INDEX idx_links_to (to_module, to_id),
    INDEX idx_links_type (link_type),
    INDEX idx_links_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Индекс сквозных ссылок [module:id] (фича 1)
CREATE TABLE IF NOT EXISTS link_index (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    source_module   VARCHAR(32) NOT NULL,
    source_id       VARCHAR(36) NOT NULL,
    target_module   VARCHAR(32) NOT NULL,
    target_id       VARCHAR(36) NOT NULL,
    INDEX idx_lnk_source (source_module, source_id),
    INDEX idx_lnk_target (target_module, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Фото записей (не для библиотеки, ADR-009)
CREATE TABLE IF NOT EXISTS record_images (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    module_key      VARCHAR(32) NOT NULL,
    entity_id       VARCHAR(36) NOT NULL,
    original_path   VARCHAR(512) NOT NULL,
    thumbnail_path  VARCHAR(512) NOT NULL,
    icon_path       VARCHAR(512) NOT NULL,
    is_primary      BOOLEAN DEFAULT FALSE,
    sort_order      INT DEFAULT 0,
    created_at      DATETIME(3) NOT NULL,
    deleted_at      DATETIME(3) NULL,
    INDEX idx_img_entity (module_key, entity_id),
    INDEX idx_img_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Состояние синхронизации по устройствам (CN из сертификата)
CREATE TABLE IF NOT EXISTS sync_state (
    device_id       VARCHAR(64) PRIMARY KEY,
    last_version    BIGINT NOT NULL DEFAULT 0,
    last_sync_at    DATETIME(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Комментарии и задачи
CREATE TABLE IF NOT EXISTS comments (
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
    INDEX idx_cmt_version (version),
    INDEX idx_cmt_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Библиотека (книги и видео, ADR-009)
CREATE TABLE IF NOT EXISTS library (
    id                  VARCHAR(36) PRIMARY KEY,
    library_category_id BIGINT NOT NULL,
    library_title       VARCHAR(255) NOT NULL,
    library_author      VARCHAR(255),
    library_format      ENUM('pdf','djvu','mp4','mkv') NOT NULL,
    library_file_path   VARCHAR(512) NOT NULL,
    library_cover_path  VARCHAR(512),
    library_size_bytes  BIGINT,
    version             BIGINT NOT NULL,
    created_at          DATETIME(3) NOT NULL,
    updated_at          DATETIME(3) NOT NULL,
    deleted_at          DATETIME(3) NULL,
    INDEX idx_lib_cat (library_category_id),
    INDEX idx_lib_version (version),
    INDEX idx_lib_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Модульные таблицы (шаблон MASTER.md 5.1 + специфика 5.3)
-- ============================================================

-- Растения (~50 характеристик через EAV + базовые поля)
CREATE TABLE IF NOT EXISTS plants (
    id                  VARCHAR(36) PRIMARY KEY,
    plants_name         VARCHAR(255) NOT NULL,
    plants_latin        VARCHAR(255),
    plants_category_id  BIGINT,
    plants_description  TEXT,
    version             BIGINT NOT NULL,
    created_at          DATETIME(3) NOT NULL,
    updated_at          DATETIME(3) NOT NULL,
    deleted_at          DATETIME(3) NULL,
    INDEX idx_plants_version (version),
    INDEX idx_plants_deleted (deleted_at),
    INDEX idx_plants_category (plants_category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Болезни (иерархия род-вид, группы патогенов)
CREATE TABLE IF NOT EXISTS diseases (
    id                          VARCHAR(36) PRIMARY KEY,
    diseases_name               VARCHAR(255) NOT NULL,
    diseases_parent_id          BIGINT NULL,
    diseases_is_pathogens_group BOOLEAN DEFAULT FALSE,
    diseases_symptoms           TEXT,
    version                     BIGINT NOT NULL,
    created_at                  DATETIME(3) NOT NULL,
    updated_at                  DATETIME(3) NOT NULL,
    deleted_at                  DATETIME(3) NULL,
    INDEX idx_diseases_version (version),
    INDEX idx_diseases_deleted (deleted_at),
    INDEX idx_diseases_parent (diseases_parent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Вредители (иерархия род-вид)
CREATE TABLE IF NOT EXISTS pests (
    id               VARCHAR(36) PRIMARY KEY,
    pests_name       VARCHAR(255) NOT NULL,
    pests_parent_id  BIGINT NULL,
    pests_description TEXT,
    version          BIGINT NOT NULL,
    created_at       DATETIME(3) NOT NULL,
    updated_at       DATETIME(3) NOT NULL,
    deleted_at       DATETIME(3) NULL,
    INDEX idx_pests_version (version),
    INDEX idx_pests_deleted (deleted_at),
    INDEX idx_pests_parent (pests_parent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Агрохимия (плоский список, удобрения отдельно)
CREATE TABLE IF NOT EXISTS agrochemicals (
    id                        VARCHAR(36) PRIMARY KEY,
    agrochemicals_name        VARCHAR(255) NOT NULL,
    agrochemicals_manufacturer VARCHAR(255),
    agrochemicals_form        VARCHAR(16) COMMENT 'SP, WG, EC, SC...',
    agrochemicals_is_fertilizer BOOLEAN DEFAULT FALSE,
    version                   BIGINT NOT NULL,
    created_at                DATETIME(3) NOT NULL,
    updated_at                DATETIME(3) NOT NULL,
    deleted_at                DATETIME(3) NULL,
    INDEX idx_agro_version (version),
    INDEX idx_agro_deleted (deleted_at),
    INDEX idx_agro_fertilizer (agrochemicals_is_fertilizer)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Химический состав удобрений (доработка 1)
CREATE TABLE IF NOT EXISTS fertilizer_composition (
    id               BIGINT PRIMARY KEY AUTO_INCREMENT,
    agrochemical_id  VARCHAR(36) NOT NULL,
    element          VARCHAR(64) NOT NULL COMMENT 'N, P2O5, K2O...',
    share_percent    DECIMAL(6,3) NOT NULL,
    UNIQUE KEY uq_fert_element (agrochemical_id, element),
    CONSTRAINT fk_fert_agro FOREIGN KEY (agrochemical_id)
        REFERENCES agrochemicals(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Действующие вещества (связь с агрохимией, баковые смеси)
CREATE TABLE IF NOT EXISTS active_substances (
    id                       VARCHAR(36) PRIMARY KEY,
    active_substances_name   VARCHAR(255) NOT NULL,
    active_substances_cas    VARCHAR(64),
    version                  BIGINT NOT NULL,
    created_at               DATETIME(3) NOT NULL,
    updated_at               DATETIME(3) NOT NULL,
    deleted_at               DATETIME(3) NULL,
    INDEX idx_as_version (version),
    INDEX idx_as_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Терминология (глоссарий, тултипы)
CREATE TABLE IF NOT EXISTS terminologies (
    id                                  VARCHAR(36) PRIMARY KEY,
    terminologies_name                  VARCHAR(255) NOT NULL,
    terminologies_short_description     TEXT COMMENT 'tooltip',
    terminologies_full_description      TEXT,
    version                             BIGINT NOT NULL,
    created_at                          DATETIME(3) NOT NULL,
    updated_at                          DATETIME(3) NOT NULL,
    deleted_at                          DATETIME(3) NULL,
    INDEX idx_term_version (version),
    INDEX idx_term_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Статьи (Markdown + предрендеренный HTML, доработка 3)
CREATE TABLE IF NOT EXISTS articles (
    id               VARCHAR(36) PRIMARY KEY,
    articles_name    VARCHAR(255) NOT NULL,
    articles_content LONGTEXT COMMENT 'Markdown со ссылками [module:id]',
    articles_html    LONGTEXT COMMENT 'предрендерен в админке (ADR-003)',
    version          BIGINT NOT NULL,
    created_at       DATETIME(3) NOT NULL,
    updated_at       DATETIME(3) NOT NULL,
    deleted_at       DATETIME(3) NULL,
    INDEX idx_articles_version (version),
    INDEX idx_articles_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Реестр питомника (уникальные артикулы, замкнутый модуль)
CREATE TABLE IF NOT EXISTS registry (
    id                VARCHAR(36) PRIMARY KEY,
    registry_article  VARCHAR(128) NOT NULL,
    registry_is_lost  BOOLEAN DEFAULT FALSE,
    registry_plant_id VARCHAR(36) NULL,
    version           BIGINT NOT NULL,
    created_at        DATETIME(3) NOT NULL,
    updated_at        DATETIME(3) NOT NULL,
    deleted_at        DATETIME(3) NULL,
    UNIQUE KEY uq_registry_article (registry_article),
    INDEX idx_registry_version (version),
    INDEX idx_registry_deleted (deleted_at),
    INDEX idx_registry_plant (registry_plant_id),
    CONSTRAINT fk_registry_plant FOREIGN KEY (registry_plant_id)
        REFERENCES plants(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Календарь (напоминания, повторения, замкнутый модуль)
CREATE TABLE IF NOT EXISTS calendar (
    id                    VARCHAR(36) PRIMARY KEY,
    calendar_title        VARCHAR(255) NOT NULL,
    calendar_event_at     DATETIME(3) NOT NULL,
    calendar_before_event INT COMMENT 'минут до события',
    calendar_repeat_after INT COMMENT 'дней до повтора, 0 или NULL - без повтора',
    version               BIGINT NOT NULL,
    created_at            DATETIME(3) NOT NULL,
    updated_at            DATETIME(3) NOT NULL,
    deleted_at            DATETIME(3) NULL,
    INDEX idx_calendar_version (version),
    INDEX idx_calendar_deleted (deleted_at),
    INDEX idx_calendar_event_at (calendar_event_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Служебная таблица версий схемы
CREATE TABLE IF NOT EXISTS schema_migrations (
    version         INT PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    applied_at      BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Сид: реестр из 11 модулей (MASTER.md, раздел 9)
-- ============================================================
INSERT INTO modules (id, name, type, is_active, created_at) VALUES
    ('plants',            'Растения',             'open',   TRUE, NOW(3)),
    ('diseases',          'Болезни',              'open',   TRUE, NOW(3)),
    ('pests',             'Вредители',            'open',   TRUE, NOW(3)),
    ('agrochemicals',     'Агрохимия',            'open',   TRUE, NOW(3)),
    ('active_substances', 'Действующие вещества', 'open',   TRUE, NOW(3)),
    ('terminologies',     'Терминология',         'open',   TRUE, NOW(3)),
    ('articles',          'Статьи',               'open',   TRUE, NOW(3)),
    ('registry',          'Реестр питомника',     'closed', TRUE, NOW(3)),
    ('calendar',          'Календарь',            'closed', TRUE, NOW(3)),
    ('library',           'Библиотека',           'closed', TRUE, NOW(3)),
    ('comments',          'Комментарии',          'closed', TRUE, NOW(3));