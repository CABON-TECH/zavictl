package models

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()
	if id1 == "" || id2 == "" {
		t.Fatal("GenerateID returned empty string")
	}
	if id1 == id2 {
		t.Fatal("GenerateID returned duplicate IDs")
	}
	// Basic sortability check (assuming called sequentially quickly)
	if id1 > id2 {
		t.Fatalf("GenerateID is not sortable: %s > %s", id1, id2)
	}
}

func TestResourceIdentity(t *testing.T) {
	r := &Resource{
		ID:   "123",
		Kind: "project",
	}
	if r.Identity() != "project/123" {
		t.Fatalf("Expected project/123, got %s", r.Identity())
	}
}

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		wantErr bool
	}{
		{
			name: "valid simple",
			labels: map[string]string{
				"env": "production",
			},
			wantErr: false,
		},
		{
			name: "valid with prefix",
			labels: map[string]string{
				"example.com/env": "production",
			},
			wantErr: false,
		},
		{
			name: "valid empty value",
			labels: map[string]string{
				"tier": "",
			},
			wantErr: false,
		},
		{
			name: "invalid key length",
			labels: map[string]string{
				strings.Repeat("a", 64): "value",
			},
			wantErr: true,
		},
		{
			name: "invalid value length",
			labels: map[string]string{
				"key": strings.Repeat("a", 64),
			},
			wantErr: true,
		},
		{
			name: "invalid characters in key",
			labels: map[string]string{
				"env@name": "prod",
			},
			wantErr: true,
		},
		{
			name: "invalid characters in prefix",
			labels: map[string]string{
				"example..com/env": "prod",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLabels(tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResourceValidate(t *testing.T) {
	r := &Resource{
		ID:   "123",
		Kind: "project",
		Name: "test-project",
		ResourceLabels: map[string]string{
			"env": "test",
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate returned unexpected error: %v", err)
	}

	r.ID = ""
	if err := r.Validate(); err == nil {
		t.Fatal("Validate should fail on empty ID")
	}
}
