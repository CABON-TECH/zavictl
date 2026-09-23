package kubernetes

import (
	"context"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
)

type kubernetesProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &kubernetesProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *kubernetesProvider) Name() string {
	return "kubernetes"
}

func (p *kubernetesProvider) Version() string {
	return p.version
}

func (p *kubernetesProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "runtime",
			Operations: []string{"scale", "restart", "logs", "inspect"},
		},
	}
}

func (p *kubernetesProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	return &kubernetesConnection{}, nil
}

func (p *kubernetesProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	return provider.HealthStatus{Status: "healthy", Message: "kubernetes is available"}, nil
}
