# zavictl — Open-source CLI-first DevOps Control Plane

`zavictl` is a unified DevOps orchestration platform that gives engineers one consistent interface for managing the entire software delivery and operations lifecycle. 

Instead of reimplementing underlying capabilities, `zavictl` coordinates existing tools (like GitHub, Kubernetes, Terraform, Vault, and Prometheus) via an intelligent intent-to-workflow engine.

## Getting Started

### Prerequisites
- Go 1.22+
- A local SQLite environment (used for the relational catalog and state store)

### Build the CLI
```bash
git clone https://github.com/CABON-TECH/my-test-repo-from-zavictl.git zavictl
cd zavictl
go build -o zavictl ./cmd/zavictl
```

### Global Flags
All commands support the following global flags to control behavior and context:
* `--environment [env]` - Target environment (e.g. staging, production)
* `--project [proj]` - Project context
* `--dry-run` - Simulate the command without making changes
* `--yes` - Automatically confirm prompts
* `--output [format]` - Format output (text, json, yaml)
* `--timeout [duration]` - Set a timeout (e.g. 30s, 5m)
* `--debug` / `--verbose` / `--quiet` - Adjust logging verbosity

---

## 📚 Command Reference

### Authentication (§7.1)
* `zavictl auth login` - Authenticate with a provider (e.g., GitHub, Vault).
* `zavictl auth logout` - Remove local credentials.
* `zavictl auth status` - View current authenticated contexts (tokens are securely masked).

### Project & Service Catalog (§7.2, §7.3)
* `zavictl project create [name]` - Create a new project.
* `zavictl project list` - List all projects.
* `zavictl project show [name]` - View project details.
* `zavictl project validate [name]` - Validate project configuration.
* `zavictl project archive [name]` - Archive a project.
* `zavictl project delete [name]` - Delete a project.
* `zavictl service create [name] --repo [url]` - Register a new service.
* `zavictl service list` - List all services.
* `zavictl service show [name]` - View service details.
* `zavictl service validate [name]` - Validate service configuration.
* `zavictl service dependencies [name]` - Map upstream/downstream dependencies.
* `zavictl service owner [name]` - View/set service ownership.

### Environments (§7.4)
* `zavictl environment create [name]` - Provision a new environment.
* `zavictl environment list` - List environments.
* `zavictl environment show [name]` - View environment details.
* `zavictl environment diff [env1] [env2]` - Compare configurations between environments.
* `zavictl environment lock [name]` - Lock an environment from deployments.
* `zavictl environment unlock [name]` - Unlock an environment.
* `zavictl environment promote [name]` - Promote an environment's state.
* `zavictl environment destroy [name]` - Decommission an environment.

### Source Control (§7.10)
* `zavictl repo create [name]` - Create a new repository on the remote provider.
* `zavictl repo list` - List managed repositories.
* `zavictl repo connect [url]` - Connect an existing repository to the platform.
* `zavictl repo status [name]` - Check repository sync status.
* `zavictl repo workflow run [name]` - Trigger a raw repository workflow.
* `zavictl repo workflow status [name]` - Check workflow status.

### Build & Artifacts (§7.6, §7.9)
* `zavictl build start [service]` - Trigger a CI build pipeline.
* `zavictl build list` - List recent builds.
* `zavictl build show [id]` - Show build execution details.
* `zavictl build logs [id]` - View build logs.
* `zavictl build cancel [id]` - Cancel a running build.
* `zavictl build retry [id]` - Retry a failed build.
* `zavictl build artifacts [id]` - View artifacts produced by a build.
* `zavictl artifact publish [service]` - Register a new artifact (e.g., container image).
* `zavictl artifact promote [id] --env [env]` - Tag an artifact for a higher environment.
* `zavictl artifact list` - List available artifacts.
* `zavictl artifact show [id]` - View artifact metadata and digests.
* `zavictl artifact verify [id]` - Verify cryptographic signatures/provenance.
* `zavictl artifact retention apply [policy]` - Cleanup stale artifacts.

### Deployments (§7.11)
* `zavictl deploy start [service]` - Trigger a deployment workflow.
* `zavictl deploy plan [service]` - Preview a deployment (Dry Run).
* `zavictl deploy status [id]` - Check deployment status.
* `zavictl deploy pause [id]` - Pause a rollout (e.g., Canary).
* `zavictl deploy resume [id]` - Resume a paused rollout.
* `zavictl deploy cancel [id]` - Cancel an ongoing deployment.
* `zavictl deploy rollback [service]` - Roll back a service to a previous stable state.

### Infrastructure (§7.13)
* `zavictl infra init [module]` - Initialize infrastructure state.
* `zavictl infra validate [module]` - Validate IaC configuration.
* `zavictl infra plan [module]` - Generate an execution plan.
* `zavictl infra show-plan [id]` - View a saved plan.
* `zavictl infra approve [id]` - Approve a plan for execution.
* `zavictl infra apply [module]` - Apply infrastructure changes.
* `zavictl infra drift [module]` - Detect configuration drift.
* `zavictl infra refresh [module]` - Refresh state from cloud provider.
* `zavictl infra import [module] [resource]` - Import existing resources into state.
* `zavictl infra destroy [module]` - Tear down infrastructure.

### Runtime Management (§7.14)
* `zavictl runtime list` - List running services and their health.
* `zavictl runtime inspect [service]` - View live Kubernetes/runtime state.
* `zavictl runtime scale [service] --replicas [n]` - Scale service replicas.
* `zavictl runtime restart [service]` - Cycle pods/containers.
* `zavictl runtime logs [service]` - Stream runtime logs.
* `zavictl runtime exec [service] -- [cmd]` - Execute a command inside the container.
* `zavictl runtime health [service]` - Run deep health checks.

### Database Operations (§7.15)
* `zavictl db list` - List active databases.
* `zavictl db provision [name]` - Provision a new database instance.
* `zavictl db migrate [name]` - Run schema migrations.
* `zavictl db backup [name]` - Trigger a database backup.
* `zavictl db restore [name] --from [id]` - Restore from a backup.
* `zavictl db failover [name]` - Trigger failover to a replica.

### Security & Secrets (§7.8, §7.17)
* `zavictl security scan [service]` - Run SAST/SCA/Container scans.
* `zavictl security findings [service]` - List vulnerability findings.
* `zavictl security approve [finding-id]` - Accept the risk for a finding.
* `zavictl secret list [service]` - List keys for a service.
* `zavictl secret set [service] [key]=[value]` - Write a secret securely to the vault.
* `zavictl secret inspect [service] [key]` - Read a specific secret.
* `zavictl secret delete [service] [key]` - Delete a secret.
* `zavictl secret inject [service]` - Inject secrets into the runtime environment.
* `zavictl secret rotate [service] [key]` - Trigger a secret rotation policy.

### Observability & Telemetry (§7.18)
* `zavictl observe metrics [service]` - Query Prometheus for key metrics.
* `zavictl observe logs [service]` - Fetch aggregated application logs.
* `zavictl observe traces [service]` - Query distributed traces.
* `zavictl observe dashboards [service]` - Get direct links to Grafana/monitoring dashboards.
* `zavictl observe audit` - View the immutable platform audit trail.
* `zavictl observe events` - View internal event bus history.
* `zavictl observe executions` - View the history of intent-to-workflow engine executions.

### Alerts & Incidents (§7.19)
* `zavictl alert list` - List active firing alerts.
* `zavictl alert acknowledge [id]` - Silence an alert.
* `zavictl alert resolve [id]` - Mark an alert as resolved.
* `zavictl incident create` - Declare an incident (routes to PagerDuty/Slack).
* `zavictl incident list` - View recent incidents.
* `zavictl incident show [id]` - View incident timeline and postmortem data.

### Policy & Governance (§7.21)
* `zavictl policy create [name] --file [path]` - Load a new CEL policy.
* `zavictl policy list` - List active policies.
* `zavictl policy delete [name]` - Remove a policy.
* `zavictl policy validate [path]` - Dry-run validate a policy against state.
* `zavictl policy test [path]` - Run unit tests for a policy.
* `zavictl policy violations` - List active policy violations across the platform.
* `zavictl policy exception [name]` - Request or grant an exception to a policy.

---

## Architecture
`zavictl` relies on an **Intent-to-Workflow** engine. When you run a command like `zavictl deploy start`, the CLI does not hardcode the API calls to GitHub or Kubernetes. Instead, it compiles a declarative intent (e.g., `github.cicd.trigger_workflow`) and submits it to the local engine. 

The engine passes the intent through:
1. **Policy Engine (CEL)** - Verifies the action is allowed.
2. **State Store (SQLite)** - Records the execution intent.
3. **Event Bus** - Broadcasts the action for the audit trail.
4. **Adapter Runner** - Routes the generic action to the configured provider (e.g., GitHub, OpenTofu, Kubernetes, Vault).

### Supported Provider Adapters
* **GitHub**: Source control and CI/CD operations.
* **Kubernetes**: Runtime and scaling operations.
* **Terraform / OpenTofu**: Infrastructure as code execution.
* **Vault**: Secret management and encryption.
* **Prometheus**: Metrics and observability.
* **Docker**: Local runtime simulation.
