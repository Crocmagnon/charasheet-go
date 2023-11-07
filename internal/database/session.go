package database

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type DjangoSession struct {
	SessionKey  string    `db:"session_key"`
	SessionData string    `db:"session_data"`
	ExpireData  time.Time `db:"expire_date"`
}

func (s *DjangoSession) Decode() (*DjangoSessionData, error) {
	split := strings.Split(s.SessionData, ":")
	value := split[0]
	isCompressed := value[0] == '.'

	if isCompressed {
		value = value[1:]
	}

	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	var session DjangoSessionData

	if !isCompressed {
		err = json.Unmarshal(data, &session)
		if err != nil {
			return nil, fmt.Errorf("unmarshalling json: %w", err)
		}

		return &session, nil
	}

	decompressed, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompressing data: %w", err)
	}

	defer decompressed.Close()

	err = json.NewDecoder(decompressed).Decode(&session)
	if err != nil {
		return nil, fmt.Errorf("decoding json: %w", err)
	}

	return &session, nil
}

type DjangoSessionData struct {
	Preview         bool   `json:"preview"`
	AuthUserID      string `json:"_auth_user_id"`
	AuthUserBackend string `json:"_auth_user_backend"`
	AuthUserHash    string `json:"_auth_user_hash"`
}

func (db *DB) GetSession(key string) (*DjangoSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var session DjangoSession

	query := `SELECT * FROM django_session WHERE session_key = $1 AND expire_date >= datetime('now')`

	err := db.GetContext(ctx, &session, query, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &session, err
}
