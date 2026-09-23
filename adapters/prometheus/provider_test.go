package prometheus

import (
	"testing"
)

func TestPrometheusProviderCapabilities(t *testing.T) {
	p := NewProvider(nil)
	caps := p.Capabilities()
	if len(caps) != 1 {
		t.Fatalf("Expected 1 capability, got %d", len(caps))
	}
	if caps[0].Domain != "observability" {
		t.Errorf("Expected observability, got %s", caps[0].Domain)
	}

	hasQuery := false
	for _, op := range caps[0].Operations {
		if op == "query" {
			hasQuery = true
		}
	}
	if !hasQuery {
		t.Error("Expected operations to include 'query'")
	}
}
