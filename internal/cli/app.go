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
		// Read from SQLite state store using List
		records, err := store.List(ctx, "Credential")
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to list credentials: %v", err)
		}
		
		// Iterate backwards to get the most recently added credential
		for i := len(records) - 1; i >= 0; i-- {
			rec := records[i]
			if rec.Data["provider"].(string) == ref.Provider {
				return rec.Data["token"].(string), time.Now().Add(24 * time.Hour), nil
			}
		}
		
		return "", time.Time{}, fmt.Errorf("no credential found for provider %s, please run 'zavictl auth login %s'", ref.Provider, ref.Provider)
	}
	resolver := credentials.NewResolver(bus, fetchFunc)

	// Load active policies from SQLite and feed them into the CEL engine
	policyDefs, _ := loadPoliciesFromStore(context.Background(), store)
	policyEngine, _ := policy.NewCELEngine(bus, policyDefs, nil)

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

// loadPoliciesFromStore reads active Policy records from SQLite and converts them to PolicyDefinitions.
func loadPoliciesFromStore(ctx context.Context, store state.StateStore) ([]policy.PolicyDefinition, error) {
	records, err := store.List(ctx, "Policy")
	if err != nil {
		return nil, err
	}

	var defs []policy.PolicyDefinition
	for _, rec := range records {
		name, _ := rec.Data["name"].(string)
		rule, _ := rec.Data["rule"].(string)
		enforcement, _ := rec.Data["enforcement"].(string)
		errMsg, _ := rec.Data["error_message"].(string)
		status, _ := rec.Data["status"].(string)

		if status != "active" || rule == "" {
			continue
		}

		if enforcement == "" {
			enforcement = "block"
		}

		defs = append(defs, policy.PolicyDefinition{
			Name: name,
			Rules: []policy.PolicyRule{
				{
					Name:        name,
					Description: errMsg,
					Expression:  rule,
					Severity:    enforcement,
				},
			},
		})
	}
	return defs, nil
}
