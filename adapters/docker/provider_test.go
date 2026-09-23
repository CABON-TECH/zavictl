package docker

import (
	"testing"
)

func TestDockerProviderCapabilities(t *testing.T) {
	p := NewProvider(nil)
	caps := p.Capabilities()
	if len(caps) != 1 {
		t.Fatalf("Expected 1 capability, got %d", len(caps))
	}
	if caps[0].Domain != "runtime" {
		t.Errorf("Expected runtime, got %s", caps[0].Domain)
	}

	hasStart := false
	for _, op := range caps[0].Operations {
		if op == "start" {
			hasStart = true
		}
	}
	if !hasStart {
		t.Error("Expected operations to include 'start'")
	}
}
