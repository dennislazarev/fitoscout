package mariadb

import (
	"context"
	"database/sql"

	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
)

// CategoriesRepo — реализация CategoryRepository для MariaDB.
type CategoriesRepo struct {
	db *sql.DB
}

// NewCategoriesRepo создаёт новый репозиторий категорий.
func NewCategoriesRepo(db *sql.DB) *CategoriesRepo {
	return &CategoriesRepo{db: db}
}

// GetByModule возвращает все категории для модуля.
func (r *CategoriesRepo) GetByModule(ctx context.Context, moduleKey string) ([]domain.Category, error) {
	query := `SELECT id, module_key, parent_id, name, icon_path, image_path, sort_order
              FROM categories WHERE module_key = ? ORDER BY sort_order`
	rows, err := r.db.QueryContext(ctx, query, moduleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		var parentID sql.NullInt64
		var iconPath, imagePath sql.NullString

		err := rows.Scan(&c.ID, &c.ModuleKey, &parentID, &c.Name, &iconPath, &imagePath, &c.SortOrder)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			c.ParentID = &parentID.Int64
		}
		if iconPath.Valid {
			c.IconPath = &iconPath.String
		}
		if imagePath.Valid {
			c.ImagePath = &imagePath.String
		}

		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// GetByID возвращает категорию по ID.
func (r *CategoriesRepo) GetByID(ctx context.Context, id int64) (*domain.Category, error) {
	query := `SELECT id, module_key, parent_id, name, icon_path, image_path, sort_order
              FROM categories WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var c domain.Category
	var parentID sql.NullInt64
	var iconPath, imagePath sql.NullString

	err := row.Scan(&c.ID, &c.ModuleKey, &parentID, &c.Name, &iconPath, &imagePath, &c.SortOrder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if parentID.Valid {
		c.ParentID = &parentID.Int64
	}
	if iconPath.Valid {
		c.IconPath = &iconPath.String
	}
	if imagePath.Valid {
		c.ImagePath = &imagePath.String
	}

	return &c, nil
}

// GetTree возвращает иерархию категорий для модуля.
func (r *CategoriesRepo) GetTree(ctx context.Context, moduleKey string) ([]domain.Category, error) {
	// Возвращаем все категории, клиент построит дерево
	return r.GetByModule(ctx, moduleKey)
}

// Create создаёт новую категорию.
func (r *CategoriesRepo) Create(ctx context.Context, category *domain.Category) error {
	query := `INSERT INTO categories (module_key, parent_id, name, icon_path, image_path, sort_order)
              VALUES (?, ?, ?, ?, ?, ?)`

	var parentID interface{}
	if category.ParentID != nil {
		parentID = *category.ParentID
	}

	var iconPath, imagePath interface{}
	if category.IconPath != nil {
		iconPath = *category.IconPath
	}
	if category.ImagePath != nil {
		imagePath = *category.ImagePath
	}

	_, err := r.db.ExecContext(ctx, query, category.ModuleKey, parentID, category.Name, iconPath, imagePath, category.SortOrder)
	return err
}

// Update обновляет категорию.
func (r *CategoriesRepo) Update(ctx context.Context, category *domain.Category) error {
	query := `UPDATE categories SET parent_id = ?, name = ?, icon_path = ?, image_path = ?, sort_order = ?
              WHERE id = ?`

	var parentID interface{}
	if category.ParentID != nil {
		parentID = *category.ParentID
	} else {
		parentID = nil
	}

	var iconPath, imagePath interface{}
	if category.IconPath != nil {
		iconPath = *category.IconPath
	} else {
		iconPath = nil
	}
	if category.ImagePath != nil {
		imagePath = *category.ImagePath
	} else {
		imagePath = nil
	}

	_, err := r.db.ExecContext(ctx, query, parentID, category.Name, iconPath, imagePath, category.SortOrder, category.ID)
	return err
}

// Delete удаляет категорию.
func (r *CategoriesRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM categories WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

var _ repositories.CategoryRepository = (*CategoriesRepo)(nil)
