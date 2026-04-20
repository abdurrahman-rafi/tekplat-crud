package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID        int
	Nama      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, nama, email, created_at, updated_at
		FROM users
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Nama, &user.Email, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func (s *UserStore) GetByID(ctx context.Context, id int) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, nama, email, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id).Scan(&user.ID, &user.Nama, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, sql.ErrNoRows
		}
		return User{}, err
	}

	return user, nil
}

func (s *UserStore) Create(ctx context.Context, nama, email string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (nama, email)
		VALUES (?, ?)
	`, nama, email)
	return err
}

func (s *UserStore) Update(ctx context.Context, id int, nama, email string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET nama = ?, email = ?
		WHERE id = ?
	`, nama, email, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		_, err := s.GetByID(ctx, id)
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (s *UserStore) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM users
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
