# Changelog

All notable changes to **zavictl** will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

> Changes that are merged but not yet tagged as a release go here.
> Move them to a versioned section when you cut a release.

---

## [0.1.0] — 2026-09-23

### Added

#### Platform Kernel
- Resource model with ULID-based IDs, optimistic concurrency versioning, and label support
- SQLite-backed state store implementing `Get`, `Put`, `List`, `Lock`, `Unlock`, and `History`
- Async event bus with secret-field redaction, wildcard subscriptions, and full-buffer drop protection
- Credential abstraction — raw secrets never leave the resolver via the `Apply` pattern
- CEL-based policy engine with `block`/`audit` severity modes and dynamic SQLite-backed rule loading
- Parallel DAG execution engine with approval gates, timeout handling, and compensation (rollback) steps
- Workflow model with cycle detection, expression references, and versioned schema
- Multi-layer configuration system with provenance tracking per key
- Provider interface with capability declarations, idempotency keys, and structured error mapping

#### Provider Adapters
- `github` — source control and CI/CD (`create_repository`, `trigger_workflow`)
- `docker` — container runtime operations
- `opentofu` — infrastructure plan and apply
- `terraform` — infrastructure plan and apply
- `vault` — secret management integration
- `prometheus` — metrics query integration

#### CLI Commands
- `zavictl auth login [provider]` — secure interactive credential entry stored in SQLite
- `zavictl auth status` — view stored provider credentials
- `zavictl project create / list / show`
- `zavictl service create / list`
- `zavictl environment create / list`
- `zavictl repo create` — triggers a live `sourcecontrol.create_repository` workflow
- `zavictl infra plan` — triggers an `opentofu.infrastructure.plan` workflow
- `zavictl deploy start` — resolves service → repository → credential → triggers remote GitHub Actions pipeline
- `zavictl policy create / list / delete` — define and manage CEL governance rules
- `zavictl observe audit` — paginated audit trail of all platform events
- `zavictl observe executions` — workflow execution history with status and last message
- `zavictl observe events --kind` — filtered event log by type
- `zavictl workflow run / list / show` — raw workflow submission and inspection
- `zavictl state get / list` — direct state store inspection

#### Observability
- Persistent `audit_events` SQLite table populated automatically from the event bus
- Every credential resolution, policy evaluation, and workflow step produces a permanent audit record
- Execution IDs (ULIDs) printed at submission time for cross-referencing audit log entries

#### Security
- Token input uses `golang.org/x/term` to suppress terminal echo
- All event metadata is redacted through `RedactMetadata` before reaching subscribers
- `--yes` flag structurally cannot bypass policy blocks or approval gates
- Credential resolver always returns the most recently stored token for a given provider

#### Open-Source Governance
- `LICENSE` (Apache 2.0)
- `CONTRIBUTING.md` with development workflow and PR guidelines
- `SECURITY.md` with vulnerability disclosure process
- `CODE_OF_CONDUCT.md`
- `GOVERNANCE.md`
- `docs/adr/` — Architecture Decision Records

### Architecture

The canonical execution path proven end-to-end in this release:

```
Intent → Workflow → Execution Engine → Policy Check → Provider → State → Audit Event
```

`zavictl deploy start` was validated against a live GitHub repository
(`CABON-TECH/my-test-repo-from-zavictl`), successfully triggering a remote
GitHub Actions pipeline via the `workflow_dispatch` API.

---

## [0.0.1] — 2026-09-22

### Added
- Initial repository scaffold
- `go.mod`, `Makefile`, `cmd/zavictl/main.go`
- CI pipeline via GitHub Actions

---

[Unreleased]: https://github.com/CABON-TECH/zavictl/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/CABON-TECH/zavictl/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/CABON-TECH/zavictl/releases/tag/v0.0.1
