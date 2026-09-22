package config

import (
	"context"
	"strings"
	"testing"
)

func TestConfigResolverAndMerge(t *testing.T) {
	yaml1 := []byte(`
server:
  host: localhost
  port: 8080
`)
	layer1, err := ParseYAML("project", "zavictl.yaml", yaml1)
	if err != nil {
		t.Fatal(err)
	}

	yaml2 := []byte(`
server:
  port: 9090
env: prod
`)
	layer2, err := ParseYAML("environment", "env.yaml", yaml2)
	if err != nil {
		t.Fatal(err)
	}

	layer3 := ConfigLayer{
		Source: "flags",
		Values: map[string]any{
			"server": map[string]any{
				"host": "0.0.0.0",
			},
		},
	}

	resolver := NewDefaultResolver([]ConfigLayer{layer1, layer2, layer3})
	rc, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	server := rc.Values["server"].(map[string]any)
	if server["host"] != "0.0.0.0" {
		t.Errorf("expected host=0.0.0.0, got %v", server["host"])
	}
	if server["port"] != 9090 {
		t.Errorf("expected port=9090, got %v", server["port"])
	}

	val, src, err := resolver.Explain("server.host")
	if err != nil {
		t.Fatal(err)
	}
	if src != "flags" {
		t.Errorf("expected source=flags for server.host, got %s", src)
	}
	if val != "0.0.0.0" {
		t.Errorf("expected val=0.0.0.0, got %v", val)
	}

	_, src, _ = resolver.Explain("server.port")
	if src != "environment" {
		t.Errorf("expected source=environment for server.port, got %s", src)
	}

	if line := rc.LineMap["server.port"]; line != 3 {
		t.Errorf("expected server.port to be on line 3 in env.yaml, got %d", line)
	}
}

func TestConfigValidation(t *testing.T) {
	yamlData := []byte(`
server:
  port: "8080"
`)
	layer, _ := ParseYAML("project", "zavictl.yaml", yamlData)
	resolver := NewDefaultResolver([]ConfigLayer{layer})
	rc, _ := resolver.Resolve(context.Background())

	schema := `
{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"required": ["server", "env"],
	"properties": {
		"server": {
			"type": "object",
			"required": ["port"],
			"properties": {
				"port": {
					"type": "integer"
				}
			}
		},
		"env": {
			"type": "string"
		}
	}
}
`

	errs := Validate(&rc, schema)
	t.Logf("LineMap: %v", rc.LineMap)
	if len(errs) != 2 {
		t.Fatalf("expected 2 validation errors, got %d: %v", len(errs), errs)
	}

	var missingEnv, invalidPort bool
	for _, e := range errs {
		t.Logf("Error field=%s, line=%d, file=%s", e.Field, e.Line, e.File)
		if strings.Contains(e.Field, "env") {
			missingEnv = true
		}
		if e.Field == "server.port" {
			invalidPort = true
			if e.Line != 3 {
				t.Errorf("expected invalid port error to point to line 3, got %d", e.Line)
			}
			if e.File != "zavictl.yaml" {
				t.Errorf("expected file zavictl.yaml, got %s", e.File)
			}
		}
	}

	if !missingEnv {
		t.Errorf("missing error for required 'env'")
	}
	if !invalidPort {
		t.Errorf("missing error for invalid 'server.port' type")
	}
}
