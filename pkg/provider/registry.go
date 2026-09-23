package provider

import (
	"fmt"
	"sync"
)

// Registry manages the available provider adapters.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry by name.
func (r *Registry) Register(name string, p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = p
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	p, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found in registry", name)
	}
	return p, nil
}
