package opentofu

import (
	"context"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
)

type opentofuProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

// NewProvider returns a new OpenTofu provider adapter.
func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &opentofuProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *opentofuProvider) Name() string {
	return "opentofu"
}

func (p *opentofuProvider) Version() string {
	return p.version
}

func (p *opentofuProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "infrastructure",
			Operations: []string{"init", "plan", "apply", "destroy"},
		},
	}
}

func (p *opentofuProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	var cred credentials.Credential
	var err error
	
	if creds.ID != "" && p.resolver != nil {
		cred, err = p.resolver.Resolve(ctx, creds)
		if err != nil {
			return nil, err
		}
	}

	return &opentofuConnection{
		cred: cred,
	}, nil
}

func (p *opentofuProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	tfConn, ok := conn.(*opentofuConnection)
	if !ok {
		return provider.HealthStatus{Status: "unknown", Message: "invalid connection type"}, nil
	}

	err := tfConn.checkBinary(ctx)
	if err != nil {
		return provider.HealthStatus{Status: "unhealthy", Message: "tofu binary not found"}, mapError(err)
	}

	return provider.HealthStatus{Status: "healthy", Message: "tofu binary is available"}, nil
}
