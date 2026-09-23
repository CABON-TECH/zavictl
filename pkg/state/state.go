package state

import (
	"context"
	"errors"
	"reflect"
	"time"

	"zavictl/pkg/models"
)

// StateRecord represents the persisted state of a resource.
type StateRecord struct {
	Version   int
	Data      map[string]any
	UpdatedAt time.Time
}

// LockToken represents an acquired lock.
type LockToken string

// StateStore is the interface for state persistence.
type StateStore interface {
	Get(ctx context.Context, ref models.ResourceRef) (StateRecord, error)
	Put(ctx context.Context, ref models.ResourceRef, record StateRecord, expectedVersion int) error
	History(ctx context.Context, ref models.ResourceRef) ([]StateRecord, error)
	Lock(ctx context.Context, ref models.ResourceRef, holder string, ttl time.Duration) (LockToken, error)
	List(ctx context.Context, kind string) ([]StateRecord, error)
	Unlock(ctx context.Context, token LockToken) error
}

// ConflictError represents an optimistic concurrency failure or locking failure.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// ErrNotFound is returned when a state record does not exist.
var ErrNotFound = errors.New("state not found")

// DriftReport represents differences between stored platform state and actual provider state.
type DriftReport struct {
	HasDrift bool
	Message  string
}

// Reconcile compares stored state against a freshly queried provider state.
func Reconcile(stored, actual map[string]any) DriftReport {
	if !reflect.DeepEqual(stored, actual) {
		return DriftReport{
			HasDrift: true,
			Message:  "drift detected: stored state does not match actual state",
		}
	}
	return DriftReport{
		HasDrift: false,
	}
}
