package vault

import (
	"context"
	"net/http"
	"os"
	"strings"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"

	vaultapi "github.com/hashicorp/vault/api"
)

type vaultProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &vaultProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *vaultProvider) Name() string {
	return "vault"
}

func (p *vaultProvider) Version() string {
	return p.version
}

func (p *vaultProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "secretmanagement",
			Operations: []string{"read", "write", "list", "delete"},
		},
	}
}

func (p *vaultProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	config := vaultapi.DefaultConfig()

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	if creds.ID != "" && p.resolver != nil {
		cred, err := p.resolver.Resolve(ctx, creds)
		if err != nil {
			return nil, err
		}

		req, _ := http.NewRequest("GET", "http://dummy", nil)
		providerReq := &credentials.ProviderRequest{Header: req.Header}
		
		err = cred.Apply(providerReq)
		if err != nil {
			return nil, err
		}

		if auth := req.Header.Get("Authorization"); auth != "" {
			token := strings.TrimPrefix(auth, "Bearer ")
			client.SetToken(token)
		} else if token := req.Header.Get("X-Vault-Token"); token != "" {
			client.SetToken(token)
		}
	} else if os.Getenv("VAULT_TOKEN") != "" {
		client.SetToken(os.Getenv("VAULT_TOKEN"))
	}

	return &vaultConnection{
		client: client,
	}, nil
}

func (p *vaultProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	vConn, ok := conn.(*vaultConnection)
	if !ok {
		return provider.HealthStatus{Status: "unknown", Message: "invalid connection type"}, nil
	}

	health, err := vConn.client.Sys().Health()
	if err != nil {
		return provider.HealthStatus{Status: "unhealthy", Message: err.Error()}, mapError(err)
	}

	if !health.Initialized {
		return provider.HealthStatus{Status: "unhealthy", Message: "vault is not initialized"}, nil
	}
	if health.Sealed {
		return provider.HealthStatus{Status: "unhealthy", Message: "vault is sealed"}, nil
	}

	return provider.HealthStatus{Status: "healthy", Message: "vault is ready"}, nil
}
