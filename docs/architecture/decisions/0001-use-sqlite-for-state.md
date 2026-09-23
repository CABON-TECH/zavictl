# 1. Use SQLite for Local State Storage

Date: 2026-09-23

## Status

Accepted

## Context

We need a durable, local-first datastore for the control plane. The state model requires strict optimistic concurrency control (version increments) and crash resumability to support parallel DAG workflows gracefully without race conditions.

## Decision

We will use SQLite in WAL (Write-Ahead Logging) mode alongside `busy_timeout(5000)` pragmas to support high concurrent I/O throughput from our go-routine worker pool. It fulfills our "local first" mandate by not requiring users to spin up PostgreSQL or Redis instances to test the engine.

## Consequences

- **Positive:** Zero external dependencies for local execution; incredibly fast reads/writes; atomic transactions natively map to our Resource Version model.
- **Negative:** Horizontal scaling of the core engine process across multiple nodes is fundamentally capped by the SQLite file lock, meaning future hosted/distributed modes will require implementing a PostgreSQL backing adapter for `StateStore`.
