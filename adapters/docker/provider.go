package docker

import (
	"context"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
)

type dockerProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &dockerProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *dockerProvider) Name() string {
	return "docker"
}

func (p *dockerProvider) Version() string {
	return p.version
}

func (p *dockerProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "runtime",
			Operations: []string{"list", "start", "stop", "inspect"},
		},
	}
}

func (p *dockerProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	var cred credentials.Credential
	var err error

	if creds.ID != "" && p.resolver != nil {
		cred, err = p.resolver.Resolve(ctx, creds)
		if err != nil {
			return nil, err
		}
	}

	return &dockerConnection{
		cred: cred,
	}, nil
}

func (p *dockerProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	dConn, ok := conn.(*dockerConnection)
	if !ok {
		return provider.HealthStatus{Status: "unknown", Message: "invalid connection type"}, nil
	}

	err := dConn.checkBinary(ctx)
	if err != nil {
		return provider.HealthStatus{Status: "unhealthy", Message: "docker binary not found"}, mapError(err)
	}

	return provider.HealthStatus{Status: "healthy", Message: "docker binary is available"}, nil
}
