package github

import (
	"context"
	"net/http"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"

	githubapi "github.com/google/go-github/v64/github"
)

type githubProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

// NewProvider returns a new GitHub provider adapter.
func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &githubProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *githubProvider) Name() string {
	return "github"
}

func (p *githubProvider) Version() string {
	return p.version
}

func (p *githubProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "sourcecontrol",
			Operations: []string{"read_repository", "create_branch"},
		},
		{
			Domain:     "cicd",
			Operations: []string{"trigger_workflow"},
		},
	}
}

func (p *githubProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	cred, err := p.resolver.Resolve(ctx, creds)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &authTransport{
			base: http.DefaultTransport,
			cred: cred,
		},
	}

	client := githubapi.NewClient(httpClient)

	return &githubConnection{
		client: client,
	}, nil
}

func (p *githubProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	ghConn, ok := conn.(*githubConnection)
	if !ok {
		return provider.HealthStatus{Status: "unknown", Message: "invalid connection type"}, nil
	}

	_, _, err := ghConn.client.RateLimits(ctx)
	if err != nil {
		return provider.HealthStatus{Status: "unhealthy", Message: err.Error()}, mapError(err)
	}

	return provider.HealthStatus{Status: "healthy", Message: "GitHub API is reachable"}, nil
}

type authTransport struct {
	base http.RoundTripper
	cred credentials.Credential
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())

	providerReq := &credentials.ProviderRequest{Header: clone.Header}
	t.cred.Apply(providerReq)

	return t.base.RoundTrip(clone)
}
