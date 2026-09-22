package config

import (
	"context"
	"fmt"
	"strings"
)

type DefaultResolver struct {
	layers   []ConfigLayer
	resolved *ResolvedConfig
}

func NewDefaultResolver(layers []ConfigLayer) *DefaultResolver {
	return &DefaultResolver{
		layers: layers,
	}
}

func (r *DefaultResolver) Resolve(ctx context.Context) (ResolvedConfig, error) {
	rc := ResolvedConfig{
		Values:     make(map[string]any),
		Provenance: make(map[string]string),
		LineMap:    make(map[string]int),
		Files:      make(map[string]string),
	}

	for _, layer := range r.layers {
		mergeMaps("", rc.Values, layer.Values, layer, &rc)
	}

	r.resolved = &rc
	return rc, nil
}

func (r *DefaultResolver) Explain(key string) (value any, source string, err error) {
	if r.resolved == nil {
		return nil, "", fmt.Errorf("configuration not resolved yet")
	}

	parts := strings.Split(key, ".")
	var current any = r.resolved.Values
	for _, p := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, "", fmt.Errorf("key '%s' not found", key)
		}
		current, ok = m[p]
		if !ok {
			return nil, "", fmt.Errorf("key '%s' not found", key)
		}
	}

	src, ok := r.resolved.Provenance[key]
	if !ok {
		// For maps, provenance is tracked on leaves, but we might query a parent object
		// We'll return the source as "mixed" if it's an object, but if it has a direct source, return it.
		return current, "mixed", nil
	}

	return current, src, nil
}

func mergeMaps(prefix string, dst, src map[string]any, layer ConfigLayer, rc *ResolvedConfig) {
	for k, v := range src {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		vMap, vIsMap := v.(map[string]any)
		if vIsMap {
			dVal, dExists := dst[k]
			var dMap map[string]any
			var dIsMap bool
			if dExists {
				dMap, dIsMap = dVal.(map[string]any)
			}
			if !dIsMap {
				dMap = make(map[string]any)
				dst[k] = dMap
			}
			mergeMaps(path, dMap, vMap, layer, rc)
		} else {
			dst[k] = v
		}

		rc.Provenance[path] = layer.Source
		if layer.FilePath != "" {
			rc.Files[path] = layer.FilePath
		}
		if layer.NodeMap != nil {
			if line, ok := layer.NodeMap[path]; ok {
				rc.LineMap[path] = line
			}
		}
	}
}
