package vault

import (
	"testing"
)

func TestVaultProviderCapabilities(t *testing.T) {
	p := NewProvider(nil)
	caps := p.Capabilities()
	if len(caps) != 1 {
		t.Fatalf("Expected 1 capability, got %d", len(caps))
	}
	if caps[0].Domain != "secretmanagement" {
		t.Errorf("Expected secretmanagement, got %s", caps[0].Domain)
	}

	hasRead := false
	for _, op := range caps[0].Operations {
		if op == "read" {
			hasRead = true
		}
	}
	if !hasRead {
		t.Error("Expected operations to include 'read'")
	}
}
