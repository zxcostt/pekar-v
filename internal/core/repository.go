package core

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("пользователь не найден")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByLogin(ctx context.Context, login string) (*User, error) {
	query := `
		SELECT id, full_name, login, password_hash, role, is_active,
		       register_number, workshop_station, warehouse_zone_id
		FROM users
		WHERE login = $1
	`
	var u User
	err := r.db.QueryRow(ctx, query, login).Scan(
		&u.ID, &u.FullName, &u.Login, &u.PasswordHash, &u.Role, &u.IsActive,
		&u.RegisterNumber, &u.WorkshopStation, &u.WarehouseZoneID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) ListAll(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, full_name, login, password_hash, role, is_active,
		       register_number, workshop_station, warehouse_zone_id
		FROM users
		ORDER BY id
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.FullName, &u.Login, &u.PasswordHash, &u.Role, &u.IsActive,
			&u.RegisterNumber, &u.WorkshopStation, &u.WarehouseZoneID,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
