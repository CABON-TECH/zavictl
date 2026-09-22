package workflow

import (
	"fmt"
	"strings"
)

// Validate ensures a WorkflowDefinition has a supported version and forms a valid DAG.
func Validate(def *WorkflowDefinition) error {
	if !isSupportedVersion(def.Version) {
		return fmt.Errorf("unsupported workflow version '%s'", def.Version)
	}

	stepMap := make(map[string]*StepDefinition)
	for i := range def.Steps {
		step := &def.Steps[i]
		if step.Name == "" {
			return fmt.Errorf("step name cannot be empty")
		}
		if _, exists := stepMap[step.Name]; exists {
			return fmt.Errorf("duplicate step name: '%s'", step.Name)
		}
		stepMap[step.Name] = step
	}

	adj := make(map[string][]string)

	for _, step := range def.Steps {
		for _, dep := range step.DependsOn {
			if _, exists := stepMap[dep]; !exists {
				return fmt.Errorf("step '%s' depends on unknown step '%s'", step.Name, dep)
			}
			adj[step.Name] = append(adj[step.Name], dep)
		}

		for _, comp := range step.CompensationFor {
			if _, exists := stepMap[comp]; !exists {
				return fmt.Errorf("step '%s' compensates for unknown step '%s'", step.Name, comp)
			}
			adj[step.Name] = append(adj[step.Name], comp)
		}

		// Also scan inputs for references like ${{ steps.validate.outputs.foo }}
		for _, expr := range step.Inputs {
			strExpr := string(expr)
			if strings.Contains(strExpr, "${{") && strings.Contains(strExpr, "}}") {
				for knownStep := range stepMap {
					marker := fmt.Sprintf("steps.%s.", knownStep)
					if strings.Contains(strExpr, marker) {
						adj[step.Name] = append(adj[step.Name], knownStep)
						// In real execution, expression parser extracts exact step.
					}
				}
			}
		}
	}

	// Cycle detection using DFS
	visited := make(map[string]int) // 0 = unvisited, 1 = visiting, 2 = visited
	var path []string

	var dfs func(node string) error
	dfs = func(node string) error {
		visited[node] = 1
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if visited[neighbor] == 1 {
				cycle := append(path, neighbor)
				return fmt.Errorf("cycle detected: %s", strings.Join(cycle, " -> "))
			}
			if visited[neighbor] == 0 {
				if err := dfs(neighbor); err != nil {
					return err
				}
			}
		}

		visited[node] = 2
		path = path[:len(path)-1]
		return nil
	}

	for node := range stepMap {
		if visited[node] == 0 {
			if err := dfs(node); err != nil {
				return err
			}
		}
	}

	return nil
}

func isSupportedVersion(version string) bool {
	for _, v := range SupportedVersions {
		if version == v {
			return true
		}
	}
	return false
}
