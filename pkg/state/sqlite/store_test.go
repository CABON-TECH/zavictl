package sqlite

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

func setupTestStore(t *testing.T) (*SQLiteStore, func()) {
	f, err := os.CreateTemp("", "zavictl-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	store, err := NewSQLiteStore(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	return store, func() {
		store.db.Close()
		os.Remove(f.Name())
	}
}

func TestSQLiteStore_GetPut(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	ref := models.ResourceRef("project/123")

	// Get non-existent
	_, err := store.Get(ctx, ref)
	if !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Put new
	record := state.StateRecord{
		Data: map[string]any{"foo": "bar"},
	}
	err = store.Put(ctx, ref, record, 0)
	if err != nil {
		t.Fatalf("unexpected error on Put: %v", err)
	}

	// Get existing
	got, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("expected version 1, got %d", got.Version)
	}
	if got.Data["foo"] != "bar" {
		t.Errorf("expected data foo=bar, got %v", got.Data)
	}

	// Put conflict
	err = store.Put(ctx, ref, record, 0)
	var conflict *state.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ConflictError, got %v", err)
	}

	// Put update
	err = store.Put(ctx, ref, record, 1)
	if err != nil {
		t.Fatalf("unexpected error on Put update: %v", err)
	}

	// History
	history, err := store.History(ctx, ref)
	if err != nil {
		t.Fatalf("unexpected error on History: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 history records, got %d", len(history))
	}
	if history[0].Version != 1 || history[1].Version != 2 {
		t.Errorf("expected versions 1 and 2, got %d and %d", history[0].Version, history[1].Version)
	}
}

func TestSQLiteStore_Locking(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	ref := models.ResourceRef("project/456")

	token1, err := store.Lock(ctx, ref, "worker1", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error on Lock: %v", err)
	}

	// Lock conflict
	_, err = store.Lock(ctx, ref, "worker2", 100*time.Millisecond)
	var conflict *state.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ConflictError on concurrent Lock, got %v", err)
	}

	// Wait for TTL expiry
	time.Sleep(150 * time.Millisecond)

	token2, err := store.Lock(ctx, ref, "worker3", 1*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error on Lock after TTL expiry: %v", err)
	}

	// Unlock expired token
	err = store.Unlock(ctx, token1)
	if err == nil {
		t.Fatalf("expected error on unlocking expired/overwritten token")
	}

	// Unlock valid token
	err = store.Unlock(ctx, token2)
	if err != nil {
		t.Fatalf("unexpected error on Unlock: %v", err)
	}
}

func TestReconcile(t *testing.T) {
	stored := map[string]any{"a": 1, "b": "test"}
	actual := map[string]any{"a": 1, "b": "test"}

	report := state.Reconcile(stored, actual)
	if report.HasDrift {
		t.Errorf("expected no drift")
	}

	actual["b"] = "changed"
	report = state.Reconcile(stored, actual)
	if !report.HasDrift {
		t.Errorf("expected drift")
	}
}
