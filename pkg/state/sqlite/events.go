package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"zavictl/pkg/events"
)

// ensureEventsTable creates the audit_events table if it doesn't exist.
func (s *SQLiteStore) ensureEventsTable() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS audit_events (
		id          TEXT PRIMARY KEY,
		event_type  TEXT NOT NULL,
		timestamp   DATETIME NOT NULL,
		actor_id    TEXT,
		actor_type  TEXT,
		status      TEXT,
		metadata    TEXT
	)`)
	return err
}

// SaveEvent persists an event to the audit_events table.
func (s *SQLiteStore) SaveEvent(ctx context.Context, ev events.Event) error {
	metaJSON, err := json.Marshal(ev.Metadata)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO audit_events (id, event_type, timestamp, actor_id, actor_type, status, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.Type, ev.Timestamp.UTC(), ev.Actor.ID, ev.Actor.Type, ev.Status, string(metaJSON),
	)
	return err
}

// AuditEvent is a flat record returned from the events table.
type AuditEvent struct {
	ID        string
	Type      string
	Timestamp time.Time
	ActorID   string
	ActorType string
	Status    string
	Metadata  map[string]any
}

// ListEvents returns all audit events, optionally filtered by event type.
func (s *SQLiteStore) ListEvents(ctx context.Context, kind string, limit int) ([]AuditEvent, error) {
	var rows *sql.Rows
	var err error

	if limit <= 0 {
		limit = 100
	}

	if kind != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, event_type, timestamp, actor_id, actor_type, status, metadata
			 FROM audit_events WHERE event_type = ? ORDER BY timestamp DESC LIMIT ?`,
			kind, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, event_type, timestamp, actor_id, actor_type, status, metadata
			 FROM audit_events ORDER BY timestamp DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AuditEvent
	for rows.Next() {
		var ev AuditEvent
		var metaStr string
		var ts time.Time
		if err := rows.Scan(&ev.ID, &ev.Type, &ts, &ev.ActorID, &ev.ActorType, &ev.Status, &metaStr); err != nil {
			return nil, err
		}
		ev.Timestamp = ts
		if metaStr != "" {
			json.Unmarshal([]byte(metaStr), &ev.Metadata)
		}
		results = append(results, ev)
	}
	return results, rows.Err()
}
