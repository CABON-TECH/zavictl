package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zavictl/pkg/state"
)

// List retrieves all state records of a specific kind (e.g. Project, Service, Environment)
func (s *SQLiteStore) List(ctx context.Context, kind string) ([]state.StateRecord, error) {
	// The resource_ref is formatted as "Kind/ID", so we can use a LIKE query.
	query := "SELECT version, data, updated_at FROM states WHERE resource_ref LIKE ?"
	rows, err := s.db.QueryContext(ctx, query, fmt.Sprintf("%s/%%", kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []state.StateRecord

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

		results = append(results, state.StateRecord{
			Version:   version,
			Data:      data,
			UpdatedAt: updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
