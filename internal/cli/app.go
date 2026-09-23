package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"zavictl/pkg/credentials"
	"zavictl/pkg/events"
	"zavictl/pkg/execution"
	"zavictl/pkg/policy"
	"zavictl/pkg/state"
	"zavictl/pkg/state/sqlite"
	"zavictl/pkg/workflow"
)

type App struct {
	Store    state.StateStore
	Bus      events.EventBus
	Engine   execution.ExecutionEngine
	Resolver credentials.CredentialResolver
	Policy   policy.PolicyEngine
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

	// We need a proper StepRunner. For now we use a dummy one.
	runner := &dummyRunner{}
	engine := execution.NewEngine(store, runner, policyEngine)

	return &App{
		Store:    store,
		Bus:      bus,
		Engine:   engine,
		Resolver: resolver,
		Policy:   policyEngine,
	}, nil
}

type dummyRunner struct{}
func (r *dummyRunner) Run(ctx context.Context, step workflow.StepDefinition) (execution.StepResult, error) {
	fmt.Printf("[RUNNER] Executing step: %s (Action: %s)\n", step.Name, step.Action)
	time.Sleep(200 * time.Millisecond) // simulate work
	return execution.StepResult{Success: true}, nil
}
