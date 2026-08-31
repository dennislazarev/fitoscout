package mariadb

import (
	"context"
	"database/sql"
	"fmt"

	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
)

// ModulesRepo — реализация ModuleRepository для MariaDB.
type ModulesRepo struct {
	db *sql.DB
}

// NewModulesRepo создаёт новый репозиторий модулей.
func NewModulesRepo(db *sql.DB) *ModulesRepo {
	return &ModulesRepo{db: db}
}

// GetAll возвращает все модули.
func (r *ModulesRepo) GetAll(ctx context.Context) ([]domain.Module, error) {
	query := `SELECT id, name, type, is_active, created_at FROM modules ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []domain.Module
	for rows.Next() {
		var m domain.Module
		err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.IsActive, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, rows.Err()
}

// GetByID возвращает модуль по ID.
func (r *ModulesRepo) GetByID(ctx context.Context, id string) (*domain.Module, error) {
	query := `SELECT id, name, type, is_active, created_at FROM modules WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var m domain.Module
	err := row.Scan(&m.ID, &m.Name, &m.Type, &m.IsActive, &m.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("модуль не найден")
		}
		return nil, err
	}
	return &m, nil
}

// GetActive возвращает только активные модули.
func (r *ModulesRepo) GetActive(ctx context.Context) ([]domain.Module, error) {
	query := `SELECT id, name, type, is_active, created_at FROM modules WHERE is_active = TRUE ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []domain.Module
	for rows.Next() {
		var m domain.Module
		err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.IsActive, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, rows.Err()
}

var _ repositories.ModuleRepository = (*ModulesRepo)(nil)
