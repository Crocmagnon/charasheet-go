package database

import (
	"context"
	"database/sql"
	"errors"
)

type Character struct {
	ID    int    `db:"id"`
	Notes string `db:"notes"`
}

func (db *DB) GetCharacter(id int) (*Character, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var character Character

	query := `SELECT id, notes FROM character_character WHERE id = $1`

	err := db.GetContext(ctx, &character, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &character, err
}

func (db *DB) SetCharacterNotes(id int, notes string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE character_character SET notes = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, notes, id)
	return err
}
