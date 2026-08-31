package mariadb

import (
	"context"
	"database/sql"

	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
)

// DictionariesRepo — реализация DictionaryRepository для MariaDB.
type DictionariesRepo struct {
	db *sql.DB
}

// NewDictionariesRepo создаёт новый репозиторий словарей.
func NewDictionariesRepo(db *sql.DB) *DictionariesRepo {
	return &DictionariesRepo{db: db}
}

// GetAll возвращает все словари.
func (r *DictionariesRepo) GetAll(ctx context.Context) ([]domain.Dictionary, error) {
	query := `SELECT id, name, description FROM dictionaries ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dicts []domain.Dictionary
	for rows.Next() {
		var d domain.Dictionary
		err := rows.Scan(&d.ID, &d.Name, &d.Description)
		if err != nil {
			return nil, err
		}
		dicts = append(dicts, d)
	}
	return dicts, rows.Err()
}

// GetByID возвращает словарь по ID.
func (r *DictionariesRepo) GetByID(ctx context.Context, id int64) (*domain.Dictionary, error) {
	query := `SELECT id, name, description FROM dictionaries WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var d domain.Dictionary
	err := row.Scan(&d.ID, &d.Name, &d.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// GetWithItems возвращает словарь с элементами.
func (r *DictionariesRepo) GetWithItems(ctx context.Context, id int64) (*domain.Dictionary, []domain.DictionaryItem, error) {
	dict, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if dict == nil {
		return nil, nil, nil
	}

	items, err := r.getItemsByDictID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return dict, items, nil
}

func (r *DictionariesRepo) getItemsByDictID(ctx context.Context, dictID int64) ([]domain.DictionaryItem, error) {
	query := `SELECT id, dictionary_id, value, description, terminology_id, sort_order
              FROM dictionary_items WHERE dictionary_id = ? ORDER BY sort_order`
	rows, err := r.db.QueryContext(ctx, query, dictID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.DictionaryItem
	for rows.Next() {
		var item domain.DictionaryItem
		var desc sql.NullString
		var termID sql.NullInt64

		err := rows.Scan(&item.ID, &item.DictionaryID, &item.Value, &desc, &termID, &item.SortOrder)
		if err != nil {
			return nil, err
		}

		if desc.Valid {
			item.Description = &desc.String
		}
		if termID.Valid {
			item.TerminologyID = &termID.Int64
		}

		items = append(items, item)
	}
	return items, rows.Err()
}

// Create создаёт новый словарь.
func (r *DictionariesRepo) Create(ctx context.Context, dict *domain.Dictionary) error {
	query := `INSERT INTO dictionaries (name, description) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, dict.Name, dict.Description)
	return err
}

// Update обновляет словарь.
func (r *DictionariesRepo) Update(ctx context.Context, dict *domain.Dictionary) error {
	query := `UPDATE dictionaries SET name = ?, description = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, dict.Name, dict.Description, dict.ID)
	return err
}

// Delete удаляет словарь (элементы удалятся каскадом).
func (r *DictionariesRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM dictionaries WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

var _ repositories.DictionaryRepository = (*DictionariesRepo)(nil)
