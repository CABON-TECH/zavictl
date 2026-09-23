package opentofu

import (
	"testing"
)

func TestOpenTofuProviderCapabilities(t *testing.T) {
	p := NewProvider(nil)
	caps := p.Capabilities()
	if len(caps) != 1 {
		t.Fatalf("Expected 1 capability, got %d", len(caps))
	}
	if caps[0].Domain != "infrastructure" {
		t.Errorf("Expected infrastructure, got %s", caps[0].Domain)
	}
	
	hasPlan := false
	for _, op := range caps[0].Operations {
		if op == "plan" {
			hasPlan = true
		}
	}
	if !hasPlan {
		t.Error("Expected operations to include 'plan'")
	}
}
