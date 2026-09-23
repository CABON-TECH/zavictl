package sqlite

import (
	"context"
	"encoding/json"
	"strings"
)

// ExecutionSummary is a structured view of a workflow execution record.
type ExecutionSummary struct {
	ID           string
	Status       string
	WorkflowName string
	LastMessage  string
}

// ListExecutions returns all workflow execution records with their IDs parsed from resource_ref.
func (s *SQLiteStore) ListExecutions(ctx context.Context) ([]ExecutionSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT resource_ref, data FROM states WHERE resource_ref LIKE 'Execution/%' ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ExecutionSummary
	for rows.Next() {
		var ref, dataStr string
		if err := rows.Scan(&ref, &dataStr); err != nil {
			return nil, err
		}

		// Parse ID from "Execution/<ULID>"
		id := strings.TrimPrefix(ref, "Execution/")

		var data map[string]any
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			continue
		}

		status, _ := data["status"].(string)
		msg, _ := data["last_message"].(string)

		wfName := ""
		if wf, ok := data["workflow"].(map[string]any); ok {
			wfName, _ = wf["name"].(string)
		}

		results = append(results, ExecutionSummary{
			ID:           id,
			Status:       status,
			WorkflowName: wfName,
			LastMessage:  msg,
		})
	}
	return results, rows.Err()
}
