package credentials

import (
	"context"
	"net/http"
	"time"

	"zavictl/pkg/events"
	"zavictl/pkg/models"
)

// CredentialRef identifies a credential without exposing its value.
type CredentialRef struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Scope    string `json:"scope"`
}

// ProviderRequest represents an outgoing provider request that needs authentication.
type ProviderRequest struct {
	Header http.Header
}

// Credential is the interface through which secrets are safely injected.
type Credential interface {
	Apply(req *ProviderRequest) error
	ExpiresAt() time.Time
}

// CredentialResolver is the interface for obtaining and managing credentials.
type CredentialResolver interface {
	Resolve(ctx context.Context, ref CredentialRef) (Credential, error)
	Rotate(ctx context.Context, ref CredentialRef) error
	Revoke(ctx context.Context, ref CredentialRef) error
}

// Event types
const (
	EventCredentialResolved = "CredentialResolved"
	EventCredentialRotated  = "CredentialRotated"
	EventCredentialRevoked  = "CredentialRevoked"
)

// secretCredential holds the raw secret value. It is strictly unexported so no
// outside package can structurally access `rawSecret`.
type secretCredential struct {
	rawSecret string
	expiresAt time.Time
}

func (c *secretCredential) Apply(req *ProviderRequest) error {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Authorization", "Bearer "+c.rawSecret)
	return nil
}

func (c *secretCredential) ExpiresAt() time.Time {
	return c.expiresAt
}

// smartCredential wraps a secret credential with transparent refresh logic.
type smartCredential struct {
	ref      CredentialRef
	resolver *auditingResolver
	current  Credential
}

func (s *smartCredential) Apply(req *ProviderRequest) error {
	if time.Now().Add(5 * time.Second).After(s.current.ExpiresAt()) {
		newCred, err := s.resolver.resolveRaw(context.Background(), s.ref)
		if err != nil {
			return err
		}
		s.current = newCred
	}
	return s.current.Apply(req)
}

func (s *smartCredential) ExpiresAt() time.Time {
	return s.current.ExpiresAt()
}

// auditingResolver wraps a raw resolver implementation to add event auditing and smart refresh.
type auditingResolver struct {
	bus      events.EventBus
	fetchRaw func(ctx context.Context, ref CredentialRef) (string, time.Time, error)
}

// NewResolver creates a new audited CredentialResolver.
func NewResolver(bus events.EventBus, fetchRaw func(ctx context.Context, ref CredentialRef) (string, time.Time, error)) CredentialResolver {
	return &auditingResolver{
		bus:      bus,
		fetchRaw: fetchRaw,
	}
}

func (r *auditingResolver) resolveRaw(ctx context.Context, ref CredentialRef) (Credential, error) {
	val, exp, err := r.fetchRaw(ctx, ref)
	if err != nil {
		return nil, err
	}
	return &secretCredential{
		rawSecret: val,
		expiresAt: exp,
	}, nil
}

func (r *auditingResolver) Resolve(ctx context.Context, ref CredentialRef) (Credential, error) {
	raw, err := r.resolveRaw(ctx, ref)
	if err != nil {
		return nil, err
	}

	r.publishAudit(ctx, EventCredentialResolved, ref)

	return &smartCredential{
		ref:      ref,
		resolver: r,
		current:  raw,
	}, nil
}

func (r *auditingResolver) Rotate(ctx context.Context, ref CredentialRef) error {
	r.publishAudit(ctx, EventCredentialRotated, ref)
	return nil
}

func (r *auditingResolver) Revoke(ctx context.Context, ref CredentialRef) error {
	r.publishAudit(ctx, EventCredentialRevoked, ref)
	return nil
}

func (r *auditingResolver) publishAudit(ctx context.Context, eventType string, ref CredentialRef) {
	ev := events.Event{
		ID:        models.GenerateID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"id":       ref.ID,
			"provider": ref.Provider,
			"scope":    ref.Scope,
		},
	}
	_ = r.bus.Publish(ctx, ev)
}
