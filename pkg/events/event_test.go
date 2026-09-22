package events

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRedactMetadata(t *testing.T) {
	meta := map[string]any{
		"public_info": "hello",
		"password":    "supersecret",
		"api_key":     "12345",
		"nested": map[string]any{
			"deep_secret": "hidden",
			"safe":        "ok",
		},
		"list": []any{
			map[string]any{
				"token": "abc",
			},
			"normal",
		},
	}

	redacted := RedactMetadata(meta)

	if redacted["public_info"] != "hello" {
		t.Errorf("expected public_info to be untouched")
	}
	if redacted["password"] != "[REDACTED]" {
		t.Errorf("expected password to be redacted, got %v", redacted["password"])
	}
	if redacted["api_key"] != "[REDACTED]" {
		t.Errorf("expected api_key to be redacted")
	}

	nested := redacted["nested"].(map[string]any)
	if nested["deep_secret"] != "[REDACTED]" {
		t.Errorf("expected deep_secret to be redacted")
	}
	if nested["safe"] != "ok" {
		t.Errorf("expected safe to be untouched")
	}

	list := redacted["list"].([]any)
	if list[1] != "normal" {
		t.Errorf("expected list element 1 to be normal")
	}
	listMap := list[0].(map[string]any)
	if listMap["token"] != "[REDACTED]" {
		t.Errorf("expected nested map token to be redacted")
	}
}

func TestAsyncEventBus(t *testing.T) {
	bus := NewAsyncEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var received Event
	sub, err := bus.Subscribe("TestEvent", func(e Event) {
		received = e
		wg.Done()
	})
	if err != nil {
		t.Fatalf("unexpected error on subscribe: %v", err)
	}

	ctx := context.Background()
	err = bus.Publish(ctx, Event{
		Type:          "TestEvent",
		CorrelationID: "123",
		Metadata: map[string]any{
			"secret_token": "hidden",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error on publish: %v", err)
	}

	// wait for async delivery
	wg.Wait()

	if received.CorrelationID != "123" {
		t.Errorf("expected correlation id 123, got %s", received.CorrelationID)
	}
	if received.Metadata["secret_token"] != "[REDACTED]" {
		t.Errorf("expected metadata to be redacted")
	}

	// Test unsubscribe
	sub.Unsubscribe()

	// Wait a bit to ensure unsubscribe removes it
	time.Sleep(10 * time.Millisecond)

	err = bus.Publish(ctx, Event{Type: "TestEvent"})
	if err != nil {
		t.Fatalf("publish shouldn't fail even with no subs")
	}
}

func TestAsyncEventBus_FullBuffer(t *testing.T) {
	bus := NewAsyncEventBus()

	// We will block the handler to fill the buffer
	handlerBlock := make(chan struct{})
	
	count := 0
	var mu sync.Mutex

	_, err := bus.Subscribe("FloodEvent", func(e Event) {
		<-handlerBlock
		mu.Lock()
		count++
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("unexpected error on subscribe: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < DefaultBufferSize+5; i++ {
		_ = bus.Publish(ctx, Event{
			Type:          "FloodEvent",
			CorrelationID: fmt.Sprintf("%d", i),
		})
	}

	// Unblock
	close(handlerBlock)

	// Wait a short time to let channel drain
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	processed := count
	mu.Unlock()

	if processed > DefaultBufferSize+1 || processed < DefaultBufferSize-1 {
		t.Errorf("expected around %d processed events due to drop, got %d", DefaultBufferSize, processed)
	}
}
