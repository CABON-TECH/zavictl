package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
	
	githubapi "github.com/google/go-github/v64/github"
)

func TestGitHubProviderCapabilities(t *testing.T) {
	p := NewProvider(nil)
	caps := p.Capabilities()
	if len(caps) != 2 {
		t.Fatalf("Expected 2 capabilities, got %d", len(caps))
	}
	if caps[0].Domain != "sourcecontrol" {
		t.Errorf("Expected sourcecontrol, got %s", caps[0].Domain)
	}
}

func TestGitHubConnection_ReadRepository(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 123, "name": "myrepo", "full_name": "myorg/myrepo", "private": true}`))
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Create auth transport using our mock token
	httpClient := &http.Client{
		Transport: &authTransport{
			base: http.DefaultTransport,
			cred: &mockCred{"secret-token"},
		},
	}
	
	client, _ := githubapi.NewClient(httpClient).WithEnterpriseURLs(server.URL, server.URL)
	
	conn := &githubConnection{
		client: client,
	}

	op := provider.Operation{
		Action: "sourcecontrol.read_repository",
		Parameters: map[string]any{
			"owner": "myorg",
			"repo":  "myrepo",
		},
	}

	res, err := conn.Execute(context.Background(), op)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if res.Status != "success" {
		t.Fatal("Expected status=success")
	}
	
	idStr := fmt.Sprintf("%v", res.Outputs["id"])
	if idStr != "123" {
		t.Errorf("Expected id 123, got %v", res.Outputs["id"])
	}
	name, ok := res.Outputs["name"].(string)
	if !ok || name != "myrepo" {
		t.Errorf("Expected name myrepo, got %v", res.Outputs["name"])
	}
}

type mockCred struct {
	token string
}

func (m *mockCred) Apply(req *credentials.ProviderRequest) error {
	req.Header.Set("Authorization", "Bearer "+m.token)
	return nil
}

func (m *mockCred) ExpiresAt() time.Time {
	return time.Now().Add(1 * time.Hour)
}
