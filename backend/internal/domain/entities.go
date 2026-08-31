// Package domain содержит доменные модели и интерфейсы (ADR-001:
// Clean Architecture, без внешних зависимостей).
package domain

import "time"

// Identifiable — сущность с ID (UUIDv7).
type Identifiable interface {
	GetID() string
	SetID(id string)
}

// Versionable — сущность с монотонной версией для LWW.
type Versionable interface {
	GetVersion() int64
	SetVersion(version int64)
}

// Timestampable — сущность с метками времени.
type Timestampable interface {
	GetCreatedAt() time.Time
	SetCreatedAt(ts time.Time)
	GetUpdatedAt() time.Time
	SetUpdatedAt(ts time.Time)
}

// SoftDeletable — сущность с поддержкой soft-delete.
type SoftDeletable interface {
	GetDeletedAt() *time.Time
	SetDeletedAt(ts *time.Time)
	IsDeleted() bool
}

// BaseEntity — базовая структура для всех модульных сущностей.
// Встраивается в конкретные модели (Plant, Disease, ...).
type BaseEntity struct {
	ID        string     `json:"id" db:"id"`
	Version   int64      `json:"version" db:"version"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Реализация интерфейсов для BaseEntity
func (b *BaseEntity) GetID() string             { return b.ID }
func (b *BaseEntity) SetID(id string)           { b.ID = id }
func (b *BaseEntity) GetVersion() int64         { return b.Version }
func (b *BaseEntity) SetVersion(v int64)        { b.Version = v }
func (b *BaseEntity) GetCreatedAt() time.Time   { return b.CreatedAt }
func (b *BaseEntity) SetCreatedAt(t time.Time)  { b.CreatedAt = t }
func (b *BaseEntity) GetUpdatedAt() time.Time   { return b.UpdatedAt }
func (b *BaseEntity) SetUpdatedAt(t time.Time)  { b.UpdatedAt = t }
func (b *BaseEntity) GetDeletedAt() *time.Time  { return b.DeletedAt }
func (b *BaseEntity) SetDeletedAt(t *time.Time) { b.DeletedAt = t }
func (b *BaseEntity) IsDeleted() bool           { return b.DeletedAt != nil }

// Module — запись реестра модулей (таблица modules).
type Module struct {
	ID        string    `json:"id" db:"id"` // 'plants', 'diseases', ...
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"` // open | closed
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Category — категория модуля (иерархия через ParentID).
type Category struct {
	ID        int64   `json:"id" db:"id"`
	ModuleKey string  `json:"module_key" db:"module_key"`
	ParentID  *int64  `json:"parent_id,omitempty" db:"parent_id"`
	Name      string  `json:"name" db:"name"`
	IconPath  *string `json:"icon_path,omitempty" db:"icon_path"`
	ImagePath *string `json:"image_path,omitempty" db:"image_path"`
	SortOrder int     `json:"sort_order" db:"sort_order"`
}

// Dictionary — словарь значений для EAV-атрибутов типа dict/multi_dict.
type Dictionary struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
}

// DictionaryItem — элемент словаря (значение + tooltip + связь с термином).
type DictionaryItem struct {
	ID            int64   `json:"id" db:"id"`
	DictionaryID  int64   `json:"dictionary_id" db:"dictionary_id"`
	Value         string  `json:"value" db:"value"`
	Description   *string `json:"description,omitempty" db:"description"`
	TerminologyID *int64  `json:"terminology_id,omitempty" db:"terminology_id"`
	SortOrder     int     `json:"sort_order" db:"sort_order"`
}

// AttributeGroup — пользовательская группировка характеристик (доработка 4).
type AttributeGroup struct {
	ID        int64  `json:"id" db:"id"`
	ModuleKey string `json:"module_key" db:"module_key"`
	Name      string `json:"name" db:"name"`
	SortOrder int    `json:"sort_order" db:"sort_order"`
}

// AttributeDefinition — определение динамической характеристики (ADR-005).
type AttributeDefinition struct {
	ID           int64  `json:"id" db:"id"`
	ModuleKey    string `json:"module_key" db:"module_key"`
	AttrKey      string `json:"attr_key" db:"attr_key"`
	DataType     string `json:"data_type" db:"data_type"` // int|float|text|date|dict|multi_dict
	Label        string `json:"label" db:"label"`
	Tooltip      string `json:"tooltip,omitempty" db:"tooltip"`
	GroupID      *int64 `json:"group_id,omitempty" db:"group_id"`
	DictionaryID *int64 `json:"dictionary_id,omitempty" db:"dictionary_id"`
	SortOrder    int    `json:"sort_order" db:"sort_order"`
}

// AttributeValue — значение характеристики в EAV-паттерне.
type AttributeValue struct {
	EntityModule string   `json:"entity_module" db:"entity_module"`
	EntityID     string   `json:"entity_id" db:"entity_id"`
	DefinitionID int64    `json:"definition_id" db:"definition_id"`
	ValueInt     *int64   `json:"value_int,omitempty" db:"value_int"`
	ValueFloat   *float64 `json:"value_float,omitempty" db:"value_float"`
	ValueText    *string  `json:"value_text,omitempty" db:"value_text"`
	ValueDate    *string  `json:"value_date,omitempty" db:"value_date"`
	ValueDictID  *int64   `json:"value_dict_id,omitempty" db:"value_dict_id"`
	Version      int64    `json:"version" db:"version"`
}

// EntityLink — универсальная связь many-to-many между записями модулей.
type EntityLink struct {
	ID         int64      `json:"id" db:"id"`
	FromModule string     `json:"from_module" db:"from_module"`
	FromID     string     `json:"from_id" db:"from_id"`
	ToModule   string     `json:"to_module" db:"to_module"`
	ToID       string     `json:"to_id" db:"to_id"`
	LinkType   string     `json:"link_type" db:"link_type"` // related | cause | treatment ...
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// RecordImage — фото записи: оригинал + превью + иконка (ADR-009).
type RecordImage struct {
	ID            int64      `json:"id" db:"id"`
	ModuleKey     string     `json:"module_key" db:"module_key"`
	EntityID      string     `json:"entity_id" db:"entity_id"`
	OriginalPath  string     `json:"original_path" db:"original_path"`
	ThumbnailPath string     `json:"thumbnail_path" db:"thumbnail_path"`
	IconPath      string     `json:"icon_path" db:"icon_path"`
	IsPrimary     bool       `json:"is_primary" db:"is_primary"`
	SortOrder     int        `json:"sort_order" db:"sort_order"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// SyncState — состояние синхронизации устройства (CN сертификата → версия).
type SyncState struct {
	DeviceID    string     `json:"device_id" db:"device_id"`
	LastVersion int64      `json:"last_version" db:"last_version"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
}

// Comment — комментарий/задача к записи модуля (comments_type).
type Comment struct {
	BaseEntity
	ModuleKey string `json:"module_key" db:"module_key"`
	EntityID  string `json:"entity_id" db:"entity_id"`
	Type      string `json:"type" db:"comments_type"` // comment | general | task
	Text      string `json:"text" db:"comments_text"`
	Status    string `json:"status" db:"comments_status"` // new | in_progress | done
}

// Plant — растение (~50 характеристик через EAV + базовые поля).
type Plant struct {
	BaseEntity
	Name        string  `json:"plants_name" db:"plants_name"`
	Latin       *string `json:"plants_latin,omitempty" db:"plants_latin"`
	CategoryID  *int64  `json:"plants_category_id,omitempty" db:"plants_category_id"`
	Description *string `json:"plants_description,omitempty" db:"plants_description"`
}

// Disease — болезнь (иерархия род→вид, is_pathogens_group).
type Disease struct {
	BaseEntity
	Name             string  `json:"diseases_name" db:"diseases_name"`
	ParentID         *int64  `json:"diseases_parent_id,omitempty" db:"diseases_parent_id"`
	IsPathogensGroup bool    `json:"diseases_is_pathogens_group" db:"diseases_is_pathogens_group"`
	Symptoms         *string `json:"diseases_symptoms,omitempty" db:"diseases_symptoms"`
}

// Pest — вредитель (иерархия род→вид).
type Pest struct {
	BaseEntity
	Name        string  `json:"pests_name" db:"pests_name"`
	ParentID    *int64  `json:"pests_parent_id,omitempty" db:"pests_parent_id"`
	Description *string `json:"pests_description,omitempty" db:"pests_description"`
}

// Agrochemical — препарат агрохимии (SP/WG/EC/SC, удобрения отдельно).
type Agrochemical struct {
	BaseEntity
	Name         string  `json:"agrochemicals_name" db:"agrochemicals_name"`
	Manufacturer *string `json:"agrochemicals_manufacturer,omitempty" db:"agrochemicals_manufacturer"`
	Form         *string `json:"agrochemicals_form,omitempty" db:"agrochemicals_form"` // SP | WG | EC | SC ...
	IsFertilizer bool    `json:"agrochemicals_is_fertilizer" db:"agrochemicals_is_fertilizer"`
}

// FertilizerComponent — элемент химического состава удобрения (доработка 1).
type FertilizerComponent struct {
	ID             int64   `json:"id" db:"id"`
	AgrochemicalID string  `json:"agrochemical_id" db:"agrochemical_id"`
	Element        string  `json:"element" db:"element"` // N, P2O5, K2O ...
	SharePercent   float64 `json:"share_percent" db:"share_percent"`
}

// ActiveSubstance — действующее вещество (связи с агрохимией, баковые смеси).
type ActiveSubstance struct {
	BaseEntity
	Name string  `json:"active_substances_name" db:"active_substances_name"`
	CAS  *string `json:"active_substances_cas,omitempty" db:"active_substances_cas"`
}

// Terminology — термин глоссария (короткое описание используется как tooltip).
type Terminology struct {
	BaseEntity
	Name             string  `json:"terminologies_name" db:"terminologies_name"`
	ShortDescription *string `json:"terminologies_short_description,omitempty" db:"terminologies_short_description"`
	FullDescription  *string `json:"terminologies_full_description,omitempty" db:"terminologies_full_description"`
}

// Article — статья: Markdown + предрендеренный HTML (ADR-003, доработка 3).
type Article struct {
	BaseEntity
	Name    string  `json:"articles_name" db:"articles_name"`
	Content string  `json:"articles_content" db:"articles_content"`     // Markdown со ссылками [module:id]
	HTML    *string `json:"articles_html,omitempty" db:"articles_html"` // предрендерен в админке
}

// RegistryItem — позиция реестра питомника (уникальный артикул).
type RegistryItem struct {
	BaseEntity
	Article string  `json:"registry_article" db:"registry_article"`
	IsLost  bool    `json:"registry_is_lost" db:"registry_is_lost"`
	PlantID *string `json:"registry_plant_id,omitempty" db:"registry_plant_id"`
}

// CalendarEvent — событие календаря с напоминанием и повторением.
type CalendarEvent struct {
	BaseEntity
	Title       string     `json:"calendar_title" db:"calendar_title"`
	EventAt     time.Time  `json:"calendar_event_at" db:"calendar_event_at"`
	BeforeEvent int        `json:"calendar_before_event" db:"calendar_before_event"` // минут до события
	RepeatAfter int        `json:"calendar_repeat_after" db:"calendar_repeat_after"` // дней до повтора, 0 = без повтора
	DoneAt      *time.Time `json:"calendar_done_at,omitempty" db:"calendar_done_at"`
}

// LibraryItem — книга/видео библиотеки (streaming через HTTP Range).
type LibraryItem struct {
	BaseEntity
	CategoryID int64   `json:"library_category_id" db:"library_category_id"`
	Title      string  `json:"library_title" db:"library_title"`
	Author     *string `json:"library_author,omitempty" db:"library_author"`
	Format     string  `json:"library_format" db:"library_format"` // pdf | djvu | mp4 | mkv
	FilePath   string  `json:"library_file_path" db:"library_file_path"`
	CoverPath  *string `json:"library_cover_path,omitempty" db:"library_cover_path"`
	SizeBytes  *int64  `json:"library_size_bytes,omitempty" db:"library_size_bytes"`
}
