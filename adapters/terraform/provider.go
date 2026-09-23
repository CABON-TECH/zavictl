package terraform

import (
	"context"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
)

type terraformProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

// NewProvider returns a new Terraform provider adapter.
func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &terraformProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *terraformProvider) Name() string {
	return "terraform"
}

func (p *terraformProvider) Version() string {
	return p.version
}

func (p *terraformProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "infrastructure",
			Operations: []string{"init", "plan", "apply", "destroy"},
		},
	}
}

func (p *terraformProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	var cred credentials.Credential
	var err error
	
	if creds.ID != "" && p.resolver != nil {
		cred, err = p.resolver.Resolve(ctx, creds)
		if err != nil {
			return nil, err
		}
	}

	return &terraformConnection{
		cred: cred,
	}, nil
}

func (p *terraformProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	tfConn, ok := conn.(*terraformConnection)
	if !ok {
		return provider.HealthStatus{Status: "unknown", Message: "invalid connection type"}, nil
	}

	err := tfConn.checkBinary(ctx)
	if err != nil {
		return provider.HealthStatus{Status: "unhealthy", Message: "terraform binary not found"}, mapError(err)
	}

	return provider.HealthStatus{Status: "healthy", Message: "terraform binary is available"}, nil
}
