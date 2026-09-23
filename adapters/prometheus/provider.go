package prometheus

import (
	"context"
	"net/http"
	"os"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"

	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type prometheusProvider struct {
	version  string
	resolver credentials.CredentialResolver
}

func NewProvider(resolver credentials.CredentialResolver) provider.Provider {
	return &prometheusProvider{
		version:  "v0.1.0",
		resolver: resolver,
	}
}

func (p *prometheusProvider) Name() string {
	return "prometheus"
}

func (p *prometheusProvider) Version() string {
	return p.version
}

func (p *prometheusProvider) Capabilities() []provider.Capability {
	return []provider.Capability{
		{
			Domain:     "observability",
			Operations: []string{"query", "query_range"},
		},
	}
}

func (p *prometheusProvider) Connect(ctx context.Context, creds credentials.CredentialRef) (provider.Connection, error) {
	address := os.Getenv("PROMETHEUS_URL")
	if address == "" {
		address = "http://localhost:9090"
	}

	transport := http.DefaultTransport
	if creds.ID != "" && p.resolver != nil {
		cred, err := p.resolver.Resolve(ctx, creds)
		if err != nil {
			return nil, err
		}

		transport = &authTransport{
			base: transport,
			cred: cred,
		}
	}

	client, err := api.NewClient(api.Config{
		Address:      address,
		RoundTripper: transport,
	})
	if err != nil {
		return nil, err
	}

	v1api := promv1.NewAPI(client)

	return &prometheusConnection{
		api: v1api,
	}, nil
}

func (p *prometheusProvider) HealthCheck(ctx context.Context, conn provider.Connection) (provider.HealthStatus, error) {
	promConn, ok := conn.(*prometheusConnection)
	if !ok {
		return provider.HealthStatus{Status: "unknown", Message: "invalid connection type"}, nil
	}

	_, err := promConn.api.Buildinfo(ctx)
	if err != nil {
		return provider.HealthStatus{Status: "unhealthy", Message: err.Error()}, mapError(err)
	}

	return provider.HealthStatus{Status: "healthy", Message: "prometheus is reachable"}, nil
}

type authTransport struct {
	base http.RoundTripper
	cred credentials.Credential
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	providerReq := &credentials.ProviderRequest{Header: clone.Header}

	err := t.cred.Apply(providerReq)
	if err != nil {
		return nil, err
	}

	return t.base.RoundTrip(clone)
}
