package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"zavictl/adapters/docker"
	"zavictl/adapters/github"
	"zavictl/adapters/opentofu"
	"zavictl/adapters/prometheus"
	"zavictl/adapters/terraform"
	"zavictl/adapters/vault"
	
	"zavictl/pkg/credentials"
	"zavictl/pkg/events"
	"zavictl/pkg/execution"
	"zavictl/pkg/policy"
	"zavictl/pkg/provider"
	"zavictl/pkg/state"
	"zavictl/pkg/state/sqlite"
)

type App struct {
	Store    state.StateStore
	Bus      events.EventBus
	Engine   execution.ExecutionEngine
	Resolver credentials.CredentialResolver
	Policy   policy.PolicyEngine
	Registry *provider.Registry
}

func BootstrapApp() (*App, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home: %v", err)
	}

	zaviDir := filepath.Join(home, ".zavictl")
	if err := os.MkdirAll(zaviDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(zaviDir, "state.db")
	store, err := sqlite.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}

	bus := events.NewAsyncEventBus()

	fetchFunc := func(ctx context.Context, ref credentials.CredentialRef) (string, time.Time, error) {
		return "dummy_token", time.Now().Add(1 * time.Hour), nil
	}
	resolver := credentials.NewResolver(bus, fetchFunc)

	// For now, no policies
	policyEngine, _ := policy.NewCELEngine(bus, nil, nil)

	// Initialize Provider Registry
	registry := provider.NewRegistry()
	registry.Register("github", github.NewProvider(resolver))
	registry.Register("docker", docker.NewProvider(resolver))
	registry.Register("opentofu", opentofu.NewProvider(resolver))
	registry.Register("terraform", terraform.NewProvider(resolver))
	registry.Register("vault", vault.NewProvider(resolver))
	registry.Register("prometheus", prometheus.NewProvider(resolver))

	// Replace dummy runner with actual AdapterRunner
	runner := execution.NewAdapterRunner(registry, resolver)
	engine := execution.NewEngine(store, runner, policyEngine)

	return &App{
		Store:    store,
		Bus:      bus,
		Engine:   engine,
		Resolver: resolver,
		Policy:   policyEngine,
		Registry: registry,
	}, nil
}
