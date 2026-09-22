package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"zavictl/pkg/credentials"
	"zavictl/pkg/events"
	"zavictl/pkg/execution"
	"zavictl/pkg/models"
	"zavictl/pkg/observability"
	"zavictl/pkg/policy"
	"zavictl/pkg/state/sqlite"
	"zavictl/pkg/workflow"
)

type stdoutAuditLogger struct{}

func (l *stdoutAuditLogger) LogEvent(ctx context.Context, e events.Event) error {
	log.Printf("[AUDIT] Event: %s | Type: %s | Meta: %v\n", e.ID, e.Type, e.Metadata)
	return nil
}

type dummyRunner struct{}

func (r *dummyRunner) Run(ctx context.Context, step workflow.StepDefinition) (execution.StepResult, error) {
	log.Printf("[RUNNER] Executing step: %s (Action: %s)\n", step.Name, step.Action)
	time.Sleep(500 * time.Millisecond) // simulate work
	log.Printf("[RUNNER] Finished step: %s\n", step.Name)
	return execution.StepResult{Success: true}, nil
}

func main() {
	fmt.Println("=== zavictl Phase 0 Integrated Test ===")

	// 1. EventBus
	bus := events.NewAsyncEventBus()

	// 2. Observability
	logger := &stdoutAuditLogger{}
	_, err := observability.StartAuditSubscription(bus, logger)
	if err != nil {
		log.Fatalf("Failed to start audit: %v", err)
	}

	// 3. StateStore
	dbPath := "zavictl_test.db"
	_ = os.Remove(dbPath) // clean start
	store, err := sqlite.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to init state store: %v", err)
	}
	fmt.Println("[SYS] SQLite StateStore initialized")

	// 4. Credential Resolver
	fetchFunc := func(ctx context.Context, ref credentials.CredentialRef) (string, time.Time, error) {
		// This secret should be redacted by the event bus before it hits the audit log
		return "super_secret_token_123", time.Now().Add(1 * time.Hour), nil
	}
	credResolver := credentials.NewResolver(bus, fetchFunc)
	
	// 5. Policy Engine
	policies := []policy.PolicyDefinition{
		{
			Name: "global-policies",
			Rules: []policy.PolicyRule{
				{
					Name:        "allow-all",
					Description: "Just a test policy",
					Expression:  `true`, // CEL expression
					Severity:    "block",
				},
			},
		},
	}
	policyEngine, err := policy.NewCELEngine(bus, policies, nil)
	if err != nil {
		log.Fatalf("Failed to init policy engine: %v", err)
	}

	// 6. Execution Engine
	runner := &dummyRunner{}
	engine := execution.NewEngine(store, runner, policyEngine)

	// 7. Workflow
	wf := workflow.WorkflowDefinition{
		Name:    "demo-deploy",
		Version: "v1",
		Steps: []workflow.StepDefinition{
			{Name: "lint", Action: "code.lint"},
			{Name: "build", Action: "artifact.build", DependsOn: []string{"lint"}},
			{Name: "deploy", Action: "runtime.deploy", DependsOn: []string{"build"}},
		},
	}

	// Execute credential resolution to show redaction working
	log.Println("[SYS] Triggering credential fetch (to demonstrate redaction)...")
	ctx := context.Background()
	_, _ = credResolver.Resolve(ctx, credentials.CredentialRef{ID: "cred-123", Provider: "github"})

	// Submit Workflow
	log.Println("[SYS] Submitting workflow...")
	execID, err := engine.Submit(ctx, wf)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}
	
	log.Printf("[SYS] Workflow submitted with Execution ID: %s\n", execID)

	// Monitor execution status
	for {
		status, err := engine.Status(ctx, execID)
		if err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}
		
		log.Printf("[SYS] Current Status: %s\n", status)
		if status == execution.StatusCompleted || status == execution.StatusFailed {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Let pending logs flush
	time.Sleep(100 * time.Millisecond)

	// Read state record manually
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", execID))
	rec, _ := store.Get(ctx, ref)
	fmt.Printf("\n=== Final State Record in SQLite ===\n")
	fmt.Printf("Version: %d\n", rec.Version)
	fmt.Printf("Data:    %v\n", rec.Data)
	fmt.Println("====================================")
}
