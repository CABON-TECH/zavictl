package config

import (
	"gopkg.in/yaml.v3"
)

// ParseYAML parses YAML content into a ConfigLayer, extracting line numbers.
func ParseYAML(source string, filePath string, content []byte) (ConfigLayer, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return ConfigLayer{}, err
	}

	var values map[string]any
	if err := yaml.Unmarshal(content, &values); err != nil {
		return ConfigLayer{}, err
	}

	nodeMap := make(map[string]int)
	if len(node.Content) > 0 {
		buildNodeMap("", node.Content[0], nodeMap)
	}

	return ConfigLayer{
		Source:   source,
		Values:   values,
		FilePath: filePath,
		NodeMap:  nodeMap,
	}, nil
}

func buildNodeMap(prefix string, node *yaml.Node, nodeMap map[string]int) {
	if node.Kind != yaml.MappingNode {
		return
	}
	
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		
		path := keyNode.Value
		if prefix != "" {
			path = prefix + "." + keyNode.Value
		}
		
		nodeMap[path] = keyNode.Line
		
		if valNode.Kind == yaml.MappingNode {
			buildNodeMap(path, valNode, nodeMap)
		}
	}
}
