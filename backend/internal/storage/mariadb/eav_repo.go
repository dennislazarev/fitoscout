package mariadb

import (
        "context"
        "database/sql"

        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// EAVRepo — реализация EAVRepository для MariaDB.
type EAVRepo struct {
        db *sql.DB
}

// NewEAVRepo создаёт новый EAV-репозиторий.
func NewEAVRepo(db *sql.DB) *EAVRepo {
        return &EAVRepo{db: db}
}

// GetDefinitionsByModule возвращает все определения характеристик для модуля.
func (r *EAVRepo) GetDefinitionsByModule(ctx context.Context, moduleKey string) ([]domain.AttributeDefinition, error) {
        query := `SELECT id, module_key, attr_key, data_type, label, tooltip, group_id, dictionary_id, sort_order
              FROM attribute_definitions WHERE module_key = ? ORDER BY sort_order`
        rows, err := r.db.QueryContext(ctx, query, moduleKey)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var defs []domain.AttributeDefinition
        for rows.Next() {
                var def domain.AttributeDefinition
                var groupID, dictID sql.NullInt64
                var tooltip sql.NullString

                err := rows.Scan(&def.ID, &def.ModuleKey, &def.AttrKey, &def.DataType, &def.Label, &tooltip, &groupID, &dictID, &def.SortOrder)
                if err != nil {
                        return nil, err
                }

                if tooltip.Valid {
                        def.Tooltip = tooltip.String
                }
                if groupID.Valid {
                        def.GroupID = &groupID.Int64
                }
                if dictID.Valid {
                        def.DictionaryID = &dictID.Int64
                }

                defs = append(defs, def)
        }
        return defs, rows.Err()
}

// CreateDefinition создаёт новое определение характеристики.
func (r *EAVRepo) CreateDefinition(ctx context.Context, def *domain.AttributeDefinition) error {
        query := `INSERT INTO attribute_definitions
              (module_key, attr_key, data_type, label, tooltip, group_id, dictionary_id, sort_order)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

        var tooltip, groupID, dictID interface{}
        if def.Tooltip != "" {
                tooltip = def.Tooltip
        }
        if def.GroupID != nil {
                groupID = *def.GroupID
        }
        if def.DictionaryID != nil {
                dictID = *def.DictionaryID
        }

        _, err := r.db.ExecContext(ctx, query, def.ModuleKey, def.AttrKey, def.DataType, def.Label, tooltip, groupID, dictID, def.SortOrder)
        return err
}

// UpdateDefinition обновляет определение характеристики.
func (r *EAVRepo) UpdateDefinition(ctx context.Context, def *domain.AttributeDefinition) error {
        query := `UPDATE attribute_definitions
              SET attr_key = ?, data_type = ?, label = ?, tooltip = ?, group_id = ?, dictionary_id = ?, sort_order = ?
              WHERE id = ?`

        var tooltip, groupID, dictID interface{}
        if def.Tooltip != "" {
                tooltip = def.Tooltip
        }
        if def.GroupID != nil {
                groupID = *def.GroupID
        } else {
                groupID = nil
        }
        if def.DictionaryID != nil {
                dictID = *def.DictionaryID
        } else {
                dictID = nil
        }

        _, err := r.db.ExecContext(ctx, query, def.AttrKey, def.DataType, def.Label, tooltip, groupID, dictID, def.SortOrder, def.ID)
        return err
}

// DeleteDefinition удаляет определение характеристики.
func (r *EAVRepo) DeleteDefinition(ctx context.Context, id int64) error {
        query := `DELETE FROM attribute_definitions WHERE id = ?`
        _, err := r.db.ExecContext(ctx, query, id)
        return err
}

// GetValuesForEntity возвращает все значения характеристик для сущности.
func (r *EAVRepo) GetValuesForEntity(ctx context.Context, moduleKey, entityID string) ([]domain.AttributeValue, error) {
        query := `SELECT entity_module, entity_id, definition_id, value_int, value_float, value_text, value_date, value_dict_id, version
              FROM attribute_values WHERE entity_module = ? AND entity_id = ?`
        rows, err := r.db.QueryContext(ctx, query, moduleKey, entityID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var values []domain.AttributeValue
        for rows.Next() {
                var val domain.AttributeValue
                var vInt sql.NullInt64
                var vFloat sql.NullFloat64
                var vText sql.NullString
                var vDate sql.NullString
                var vDictID sql.NullInt64

                err := rows.Scan(&val.EntityModule, &val.EntityID, &val.DefinitionID, &vInt, &vFloat, &vText, &vDate, &vDictID, &val.Version)
                if err != nil {
                        return nil, err
                }

                if vInt.Valid {
                        val.ValueInt = &vInt.Int64
                }
                if vFloat.Valid {
                        val.ValueFloat = &vFloat.Float64
                }
                if vText.Valid {
                        val.ValueText = &vText.String
                }
                if vDate.Valid {
                        val.ValueDate = &vDate.String
                }
                if vDictID.Valid {
                        val.ValueDictID = &vDictID.Int64
                }

                values = append(values, val)
        }
        return values, rows.Err()
}

// SetValue устанавливает значение характеристики (INSERT или UPDATE).
func (r *EAVRepo) SetValue(ctx context.Context, value *domain.AttributeValue) error {
        query := `INSERT INTO attribute_values
              (entity_module, entity_id, definition_id, value_int, value_float, value_text, value_date, value_dict_id, version)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
              ON DUPLICATE KEY UPDATE
                  value_int = VALUES(value_int),
                  value_float = VALUES(value_float),
                  value_text = VALUES(value_text),
                  value_date = VALUES(value_date),
                  value_dict_id = VALUES(value_dict_id),
                  version = VALUES(version)`

        var vInt, vDictID interface{}
        var vFloat interface{}
        var vText, vDate interface{}

        if value.ValueInt != nil {
                vInt = *value.ValueInt
        }
        if value.ValueFloat != nil {
                vFloat = *value.ValueFloat
        }
        if value.ValueText != nil {
                vText = *value.ValueText
        }
        if value.ValueDate != nil {
                vDate = *value.ValueDate
        }
        if value.ValueDictID != nil {
                vDictID = *value.ValueDictID
        }

        _, err := r.db.ExecContext(ctx, query, value.EntityModule, value.EntityID, value.DefinitionID, vInt, vFloat, vText, vDate, vDictID, value.Version)
        return err
}

// DeleteValuesForEntity удаляет все значения характеристик для сущности.
func (r *EAVRepo) DeleteValuesForEntity(ctx context.Context, moduleKey, entityID string) error {
        query := `DELETE FROM attribute_values WHERE entity_module = ? AND entity_id = ?`
        _, err := r.db.ExecContext(ctx, query, moduleKey, entityID)
        return err
}

var _ repositories.EAVRepository = (*EAVRepo)(nil)