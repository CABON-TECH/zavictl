package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validate validates the resolved config against a JSON schema string.
func Validate(rc *ResolvedConfig, schemaStr string) []ConfigError {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(schemaStr)); err != nil {
		return []ConfigError{{Problem: fmt.Sprintf("invalid schema: %v", err)}}
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return []ConfigError{{Problem: fmt.Sprintf("failed to compile schema: %v", err)}}
	}

	dataBytes, _ := json.Marshal(rc.Values)
	var v any
	json.Unmarshal(dataBytes, &v)

	err = schema.Validate(v)
	if err == nil {
		return nil
	}

	var errors []ConfigError

	if valErr, ok := err.(*jsonschema.ValidationError); ok {
		errors = extractErrors(rc, valErr)
	} else {
		errors = append(errors, ConfigError{Problem: err.Error()})
	}

	return errors
}

func extractErrors(rc *ResolvedConfig, valErr *jsonschema.ValidationError) []ConfigError {
	if len(valErr.Causes) == 0 {
		return []ConfigError{buildConfigError(rc, valErr)}
	}

	var errs []ConfigError
	for _, cause := range valErr.Causes {
		errs = append(errs, extractErrors(rc, cause)...)
	}
	return errs
}

func buildConfigError(rc *ResolvedConfig, valErr *jsonschema.ValidationError) ConfigError {
	path := strings.TrimPrefix(valErr.InstanceLocation, "/")
	path = strings.ReplaceAll(path, "/", ".")

	// If the error is about a missing required property (e.g. at the root),
	// the path is the parent, and the message says "missing properties: 'foo'".
	// We can try to extract 'foo' to improve the path.
	fieldPath := path
	if strings.Contains(valErr.Message, "missing properties") {
		// naive extraction
		parts := strings.Split(valErr.Message, "'")
		if len(parts) >= 3 {
			if fieldPath != "" {
				fieldPath += "." + parts[1]
			} else {
				fieldPath = parts[1]
			}
		}
	}

	ce := ConfigError{
		Field:   fieldPath,
		Problem: valErr.Message,
	}

	if rc != nil {
		if file, ok := rc.Files[fieldPath]; ok {
			ce.File = file
		}
		if line, ok := rc.LineMap[fieldPath]; ok {
			ce.Line = line
		}
	}

	return ce
}
