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
	repo := &Repository{db: db}
	repo.ensureHashedPasswords(context.Background())
	return repo
}

func (r *Repository) ensureHashedPasswords(ctx context.Context) {
	rows, err := r.db.Query(ctx, `SELECT id, password_hash FROM users WHERE password_hash NOT LIKE '$2%'`)
	if err != nil {
		return
	}
	defer rows.Close()

	type toUpdate struct {
		id   int
		hash string
	}
	var updates []toUpdate

	for rows.Next() {
		var id int
		var plain string
		if err := rows.Scan(&id, &plain); err == nil {
			if hashed, err := HashPassword(plain); err == nil {
				updates = append(updates, toUpdate{id: id, hash: hashed})
			}
		}
	}
	rows.Close()

	for _, u := range updates {
		_, _ = r.db.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, u.hash, u.id)
	}
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, userID int, newHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, newHash, userID)
	return err
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

func (r *Repository) GetByID(ctx context.Context, id int) (*User, error) {
	query := `
		SELECT id, full_name, login, password_hash, role, is_active,
		       register_number, workshop_station, warehouse_zone_id
		FROM users
		WHERE id = $1
	`
	var u User
	err := r.db.QueryRow(ctx, query, id).Scan(
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

func (r *Repository) HardDeleteUser(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) DeactivateUser(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET is_active = FALSE WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
