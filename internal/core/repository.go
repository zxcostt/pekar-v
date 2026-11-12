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

func (r *Repository) CreateUser(ctx context.Context, u *User) (*User, error) {
	query := `
		INSERT INTO users (full_name, login, password_hash, role, is_active,
		                   register_number, workshop_station, warehouse_zone_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, full_name, login, password_hash, role, is_active,
		          register_number, workshop_station, warehouse_zone_id
	`
	var created User
	err := r.db.QueryRow(ctx, query,
		u.FullName,
		u.Login,
		u.PasswordHash,
		u.Role,
		u.IsActive,
		u.RegisterNumber,
		u.WorkshopStation,
		u.WarehouseZoneID,
	).Scan(
		&created.ID,
		&created.FullName,
		&created.Login,
		&created.PasswordHash,
		&created.Role,
		&created.IsActive,
		&created.RegisterNumber,
		&created.WorkshopStation,
		&created.WarehouseZoneID,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}
