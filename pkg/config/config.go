package config

import (
	"context"
	"fmt"
)

// ConfigLayer represents a single layer of configuration in the precedence chain.
type ConfigLayer struct {
	Source   string
	Values   map[string]any
	FilePath string
	NodeMap  map[string]int // key path -> line number
}

// ResolvedConfig represents the fully merged configuration and its provenance.
type ResolvedConfig struct {
	Values     map[string]any
	Provenance map[string]string // key path -> source
	LineMap    map[string]int    // key path -> line number
	Files      map[string]string // key path -> file path
}

// ConfigResolver resolves effective configuration from multiple layers.
type ConfigResolver interface {
	Resolve(ctx context.Context) (ResolvedConfig, error)
	Explain(key string) (value any, source string, err error)
}

// ConfigError represents a structured configuration validation error.
type ConfigError struct {
	File                string
	Line                int
	Field               string
	Problem             string
	ExpectedValue       string
	SuggestedCorrection string
}

func (e *ConfigError) Error() string {
	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s:%d field '%s': %s (expected: %s). %s",
			e.File, e.Line, e.Field, e.Problem, e.ExpectedValue, e.SuggestedCorrection)
	}
	return fmt.Sprintf("field '%s': %s (expected: %s). %s",
		e.Field, e.Problem, e.ExpectedValue, e.SuggestedCorrection)
}
