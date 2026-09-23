package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func initSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS states (
			resource_ref TEXT PRIMARY KEY,
			version INTEGER NOT NULL,
			data TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS state_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_ref TEXT NOT NULL,
			version INTEGER NOT NULL,
			data TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS locks (
			resource_ref TEXT PRIMARY KEY,
			holder TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			token TEXT NOT NULL
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, ref models.ResourceRef) (state.StateRecord, error) {
	row := s.db.QueryRowContext(ctx, "SELECT version, data, updated_at FROM states WHERE resource_ref = ?", string(ref))

	var version int
	var dataStr string
	var updatedAt time.Time
	if err := row.Scan(&version, &dataStr, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return state.StateRecord{}, state.ErrNotFound
		}
		return state.StateRecord{}, err
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return state.StateRecord{}, err
	}

	return state.StateRecord{
		Version:   version,
		Data:      data,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *SQLiteStore) Put(ctx context.Context, ref models.ResourceRef, record state.StateRecord, expectedVersion int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, "SELECT version FROM states WHERE resource_ref = ?", string(ref))
	var currentVersion int
	err = row.Scan(&currentVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	isNew := errors.Is(err, sql.ErrNoRows)

	if !isNew && currentVersion != expectedVersion {
		return &state.ConflictError{Message: fmt.Sprintf("optimistic concurrency failure: expected version %d but got %d", expectedVersion, currentVersion)}
	}
	if isNew && expectedVersion != 0 {
		return &state.ConflictError{Message: fmt.Sprintf("optimistic concurrency failure: expected version %d but record does not exist", expectedVersion)}
	}

	dataBytes, err := json.Marshal(record.Data)
	if err != nil {
		return err
	}

	newVersion := currentVersion + 1
	now := time.Now()

	if isNew {
		_, err = tx.ExecContext(ctx, "INSERT INTO states (resource_ref, version, data, updated_at) VALUES (?, ?, ?, ?)", string(ref), newVersion, string(dataBytes), now)
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE states SET version = ?, data = ?, updated_at = ? WHERE resource_ref = ?", newVersion, string(dataBytes), now, string(ref))
	}

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO state_history (resource_ref, version, data, updated_at) VALUES (?, ?, ?, ?)", string(ref), newVersion, string(dataBytes), now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) History(ctx context.Context, ref models.ResourceRef) ([]state.StateRecord, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT version, data, updated_at FROM state_history WHERE resource_ref = ? ORDER BY version ASC", string(ref))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []state.StateRecord
	for rows.Next() {
		var version int
		var dataStr string
		var updatedAt time.Time
		if err := rows.Scan(&version, &dataStr, &updatedAt); err != nil {
			return nil, err
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			return nil, err
		}
		history = append(history, state.StateRecord{
			Version:   version,
			Data:      data,
			UpdatedAt: updatedAt,
		})
	}
	return history, rows.Err()
}

func (s *SQLiteStore) Lock(ctx context.Context, ref models.ResourceRef, holder string, ttl time.Duration) (state.LockToken, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	now := time.Now()
	
	row := tx.QueryRowContext(ctx, "SELECT holder, expires_at FROM locks WHERE resource_ref = ?", string(ref))
	var existingHolder string
	var expiresAt time.Time
	err = row.Scan(&existingHolder, &expiresAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	if !errors.Is(err, sql.ErrNoRows) && now.Before(expiresAt) {
		return "", &state.ConflictError{Message: fmt.Sprintf("locked by %s until %v", existingHolder, expiresAt)}
	}

	tokenStr := models.GenerateID()
	newExpires := now.Add(ttl)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, "INSERT INTO locks (resource_ref, holder, expires_at, token) VALUES (?, ?, ?, ?)", string(ref), holder, newExpires, tokenStr)
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE locks SET holder = ?, expires_at = ?, token = ? WHERE resource_ref = ?", holder, newExpires, tokenStr, string(ref))
	}

	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return state.LockToken(tokenStr), nil
}

func (s *SQLiteStore) Unlock(ctx context.Context, token state.LockToken) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM locks WHERE token = ?", string(token))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("invalid or expired lock token")
	}
	return nil
}
