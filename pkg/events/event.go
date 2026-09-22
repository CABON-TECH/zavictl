package events

import (
	"context"
	"log"
	"regexp"
	"sync"
	"time"

	"zavictl/pkg/models"
)

// ActorRef represents an entity that triggered an event.
type ActorRef struct {
	ID   string `json:"id"`
	Type string `json:"type"` // e.g. "user", "system", "service_account"
}

// Event represents a domain event in the platform.
type Event struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	CorrelationID string               `json:"correlation_id"`
	CausationID   string               `json:"causation_id,omitempty"`
	Actor         ActorRef             `json:"actor"`
	ResourceRefs  []models.ResourceRef `json:"resource_refs"`
	Timestamp     time.Time            `json:"timestamp"`
	Status        string               `json:"status"`
	Metadata      map[string]any       `json:"metadata"`
}

// Subscription represents an active subscription to the EventBus.
type Subscription interface {
	Unsubscribe()
}

// EventBus provides publish/subscribe capabilities for domain events.
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType string, handler func(Event)) (Subscription, error)
}

// --- Implementation ---

// DefaultBufferSize is the channel buffer size per subscriber.
const DefaultBufferSize = 100

// AsyncEventBus is an in-memory implementation of EventBus.
type AsyncEventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
}

type subscriber struct {
	id      string
	handler func(Event)
	ch      chan Event
	bus     *AsyncEventBus
	eType   string
}

type subscription struct {
	sub *subscriber
}

func (s *subscription) Unsubscribe() {
	s.sub.bus.removeSubscriber(s.sub.eType, s.sub)
}

// NewAsyncEventBus creates a new asynchronous event bus.
func NewAsyncEventBus() *AsyncEventBus {
	return &AsyncEventBus{
		subscribers: make(map[string][]*subscriber),
	}
}

// Publish publishes an event to all subscribers matching the event Type.
// Delivery is asynchronous. Full buffers drop the oldest event.
func (b *AsyncEventBus) Publish(ctx context.Context, event Event) error {
	// 1. Redact metadata before it leaves the publishing component
	event.Metadata = RedactMetadata(event.Metadata)

	if event.ID == "" {
		event.ID = models.GenerateID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	b.mu.RLock()
	subs := b.subscribers[event.Type]
	wildcardSubs := b.subscribers["*"]
	b.mu.RUnlock()

	allSubs := append(subs, wildcardSubs...)
	for _, sub := range allSubs {
		select {
		case sub.ch <- event:
			// successfully queued
		default:
			// buffer full: drop oldest
			select {
			case <-sub.ch:
				log.Printf("WARN: subscriber buffer full, dropping oldest event (type=%s, id=%s)", event.Type, event.ID)
			default:
			}
			// try to queue again
			select {
			case sub.ch <- event:
			default:
				log.Printf("WARN: failed to queue event even after drop (type=%s, id=%s)", event.Type, event.ID)
			}
		}
	}
	return nil
}

// Subscribe registers a handler for a specific event type.
func (b *AsyncEventBus) Subscribe(eventType string, handler func(Event)) (Subscription, error) {
	sub := &subscriber{
		id:      models.GenerateID(),
		handler: handler,
		ch:      make(chan Event, DefaultBufferSize),
		bus:     b,
		eType:   eventType,
	}

	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], sub)
	b.mu.Unlock()

	// start worker for this subscriber
	go func() {
		for ev := range sub.ch {
			sub.handler(ev)
		}
	}()

	return &subscription{sub: sub}, nil
}

func (b *AsyncEventBus) removeSubscriber(eventType string, subToRemove *subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[eventType]
	for i, s := range subs {
		if s.id == subToRemove.id {
			b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			close(s.ch)
			break
		}
	}
}

// --- Redaction ---

// secretPatterns represents keys that should be redacted.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)key`),
	regexp.MustCompile(`(?i)credential`),
	regexp.MustCompile(`(?i)auth`),
}

// RedactMetadata deeply traverses a metadata map and replaces values of keys matching secret patterns with "[REDACTED]".
func RedactMetadata(meta map[string]any) map[string]any {
	if meta == nil {
		return nil
	}
	redacted := make(map[string]any)
	for k, v := range meta {
		if isSecretKey(k) {
			redacted[k] = "[REDACTED]"
			continue
		}
		
		switch typedVal := v.(type) {
		case map[string]any:
			redacted[k] = RedactMetadata(typedVal)
		case []any:
			redacted[k] = redactSlice(typedVal)
		default:
			redacted[k] = v
		}
	}
	return redacted
}

func redactSlice(s []any) []any {
	res := make([]any, len(s))
	for i, v := range s {
		switch typedVal := v.(type) {
		case map[string]any:
			res[i] = RedactMetadata(typedVal)
		case []any:
			res[i] = redactSlice(typedVal)
		default:
			res[i] = v
		}
	}
	return res
}

func isSecretKey(k string) bool {
	for _, p := range secretPatterns {
		if p.MatchString(k) {
			return true
		}
	}
	return false
}
