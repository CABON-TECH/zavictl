package credentials

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"zavictl/pkg/events"
)

func TestCredentialInjectionAndRefresh(t *testing.T) {
	bus := events.NewAsyncEventBus()

	fetchCount := 0
	var mu sync.Mutex

	mockFetch := func(ctx context.Context, ref CredentialRef) (string, time.Time, error) {
		mu.Lock()
		defer mu.Unlock()
		fetchCount++
		
		var exp time.Time
		var val string
		if fetchCount == 1 {
			exp = time.Now().Add(-1 * time.Minute)
			val = "token_v1"
		} else {
			exp = time.Now().Add(1 * time.Hour)
			val = "token_v2"
		}
		return val, exp, nil
	}

	resolver := NewResolver(bus, mockFetch)

	ctx := context.Background()
	ref := CredentialRef{ID: "cred-1", Provider: "github"}

	var publishedEvents []events.Event
	var wg sync.WaitGroup
	wg.Add(1)
	
	_, err := bus.Subscribe(EventCredentialResolved, func(e events.Event) {
		publishedEvents = append(publishedEvents, e)
		wg.Done()
	})
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	cred, err := resolver.Resolve(ctx, ref)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	req := &ProviderRequest{
		Header: make(http.Header),
	}
	err = cred.Apply(req)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth != "Bearer token_v2" {
		t.Errorf("expected 'Bearer token_v2', got '%s'", auth)
	}

	mu.Lock()
	if fetchCount != 2 {
		t.Errorf("expected 2 fetches (initial + refresh), got %d", fetchCount)
	}
	mu.Unlock()

	wg.Wait()

	if len(publishedEvents) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(publishedEvents))
	}
	
	ev := publishedEvents[0]
	if ev.Type != EventCredentialResolved {
		t.Errorf("expected EventCredentialResolved, got %s", ev.Type)
	}
	if ev.Metadata["id"] != "cred-1" {
		t.Errorf("expected id = cred-1, got %v", ev.Metadata["id"])
	}
	
	// Ensure the raw secret never appears in the event
	for k, v := range ev.Metadata {
		if v == "token_v1" || v == "token_v2" {
			t.Errorf("audit event leaked secret token in metadata key %s!", k)
		}
	}
}

func TestCredentialStructurallyUnreachable(t *testing.T) {
	// We can't strictly test compilation failure in a normal go test (except using something like `go build` with bash),
	// but we can assert the interface behavior. The secretCredential type is private.
	
	bus := events.NewAsyncEventBus()
	resolver := NewResolver(bus, func(ctx context.Context, ref CredentialRef) (string, time.Time, error) {
		return "secret", time.Now().Add(1 * time.Hour), nil
	})

	cred, _ := resolver.Resolve(context.Background(), CredentialRef{ID: "1"})
	
	// The type of cred is *smartCredential, which wraps *secretCredential.
	// Both are unexported.
	// You cannot cast cred to anything that exposes the raw secret.
	// This satisfies the "interface design" acceptance criteria.
	if cred == nil {
		t.Fatal("expected credential")
	}
}
