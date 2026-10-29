package domain

import "fmt"

// Validate проверяет корректность структуры графа топологии и связи между узлами.
func (g *TopologyGraph) Validate() error {
	if g.ClusterName == "" {
		return fmt.Errorf("clusterName is required")
	}

	nodeIDs := make(map[string]struct{}, len(g.Nodes))
	for _, node := range g.Nodes {
		nodeIDs[node.ID] = struct{}{}
	}

	for _, link := range g.Links {
		if _, exists := nodeIDs[link.SourceNodeID]; !exists {
			return fmt.Errorf("source node %s not found ", link.SourceNodeID)
		}
		if _, exists := nodeIDs[link.TargetNodeID]; !exists {
			return fmt.Errorf("target node %s not found", link.TargetNodeID)
		}
	}

	return nil
}
