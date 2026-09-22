package observability

import (
	"context"
	"log"

	"zavictl/pkg/events"
)

// StartAuditSubscription binds an AuditLogger to an EventBus via wildcard subscription.
func StartAuditSubscription(bus events.EventBus, logger AuditLogger) (events.Subscription, error) {
	sub, err := bus.Subscribe("*", func(e events.Event) {
		// Event metadata is already permanently redacted by the EventBus during Publish.
		// Log the event to the configured sink.
		if err := logger.LogEvent(context.Background(), e); err != nil {
			log.Printf("Failed to audit log event %s: %v", e.ID, err)
		}
	})
	
	return sub, err
}
