package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID             int       `db:"id"`
	Created        time.Time `db:"date_joined"`
	Email          string    `db:"email"`
	HashedPassword string    `db:"password"`
}

func (db *DB) InsertUser(email, hashedPassword string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		INSERT INTO common_user (date_joined, username, email, password, is_active, first_name, last_name, is_superuser, is_staff)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	result, err := db.ExecContext(ctx, query, time.Now(), email, email, hashedPassword, true, "", "", false, false)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), err
}

func (db *DB) GetUser(id int) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user User

	query := `SELECT id, date_joined, email, password FROM common_user WHERE id = $1`

	err := db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &user, err
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user User

	query := `SELECT id, date_joined, email, password FROM common_user WHERE email = $1`

	err := db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &user, err
}

func (db *DB) UpdateUserHashedPassword(id int, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE common_user SET password = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, hashedPassword, id)
	return err
}
