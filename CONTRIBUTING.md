# Contributing to zavictl

Thank you for investing your time in contributing to our project!

## Development Process
1. Fork the repo and create your branch from `main`.
2. Ensure you have Go 1.21+ installed.
3. If you've added code that should be tested, add tests.
4. If you've changed APIs, update the documentation.
5. Ensure the test suite passes (`make check`).
6. Make sure your code lints (`make lint`).
7. Issue that pull request!

## Adding a New Provider Adapter
When adding a new provider adapter in `adapters/`:
1. Ensure the provider struct implements `provider.Provider`.
2. Ensure the connection struct implements `provider.Connection`.
3. Provide a constructor `NewProvider(resolver credentials.CredentialResolver)`.
4. Register the new provider inside `internal/cli/app.go` within the `BootstrapApp()` initialization loop.

## Code of Conduct
Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.
