package topology_test

import (
	"DBSentinel/internal/app/providers"
	"DBSentinel/internal/app/topology"
	"DBSentinel/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testClusterName = "production-cluster"
	testLsn         = "0/3000060"
)

type mockConnector struct {
	mu        sync.Mutex
	responses map[string]*topology.NodeInspectionResult
	errs      map[string]error
	calls     map[string]int
}

func newMockConnector() *mockConnector {
	return &mockConnector{
		responses: make(map[string]*topology.NodeInspectionResult),
		errs:      make(map[string]error),
		calls:     make(map[string]int),
	}
}

func (m *mockConnector) setResponse(endpoint string, res *topology.NodeInspectionResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[endpoint] = res
}

func (m *mockConnector) setError(endpoint string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errs[endpoint] = err
}

func (m *mockConnector) InspectNode(ctx context.Context, target domain.TargetEndpoint) (*topology.NodeInspectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := target.String()
	m.calls[key]++

	if err, ok := m.errs[key]; ok {
		return nil, err
	}

	if res, ok := m.responses[key]; ok {
		return res, nil
	}

	return nil, errors.New("node not found")
}

func TestCollector_Collect_Success(t *testing.T) {
	ctx := context.Background()

	primaryEndpoint := domain.TargetEndpoint{Host: "master.db", Port: 5432}
	replicaEndpoint := domain.TargetEndpoint{Host: "replica.db", Port: 5433}

	connector := newMockConnector()

	// Настройка ответа для Primary
	connector.setResponse(primaryEndpoint.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       primaryEndpoint.String(),
			Hostname: primaryEndpoint.Host,
			Role:     domain.RolePrimary,
			Status:   domain.StatusHealthy,
			Version:  "PostgreSQL 18.0",
			ReadOnly: false,
		},
		ReplicationLinks: []domain.ReplicationLink{
			{
				ID:           "repl-master-replica",
				SourceNodeID: primaryEndpoint.String(),
				TargetNodeID: replicaEndpoint.String(),
				Type:         domain.TypeAsync,
				Status:       domain.ChannelStatusActive,
				PositionDetails: domain.PositionDetails{
					SourceLsn:  testLsn,
					AppliedLsn: testLsn,
					SentLsn:    testLsn,
				},
			},
		},
		DiscoveredPeers: []domain.TargetEndpoint{replicaEndpoint},
	})

	// Настройка ответа для Replica
	connector.setResponse(replicaEndpoint.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       replicaEndpoint.String(),
			Hostname: replicaEndpoint.Host,
			Role:     domain.RoleReplica,
			Status:   domain.StatusHealthy,
			Version:  "PostgreSQL 18.0",
			ReadOnly: true,
			Metrics: domain.NodeMetrics{
				ReplicationLagBytes: 0,
			},
		},
		ReplicationLinks: nil,
		DiscoveredPeers:  nil,
	})

	seedProvider := providers.NewStaticSeedProvider(primaryEndpoint)
	collector := topology.NewCollector(
		seedProvider,
		topology.WithClusterName(testClusterName),
		topology.WithConnector(connector),
		topology.WithTimeout(5*time.Second),
	)

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected collect error: %v", err)
	}

	if graph.ClusterName != testClusterName {
		t.Errorf("expected clusterName %q, got %q", testClusterName, graph.ClusterName)
	}

	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes in graph, got %d", len(graph.Nodes))
	}

	if len(graph.Links) != 1 {
		t.Fatalf("expected 1 link in graph, got %d", len(graph.Links))
	}

	if err := graph.Validate(); err != nil {
		t.Errorf("graph validation failed: %v", err)
	}
}

func TestCollector_Collect_PartialFailure(t *testing.T) {
	ctx := context.Background()

	node1 := domain.TargetEndpoint{Host: "node1", Port: 5432}
	node2 := domain.TargetEndpoint{Host: "node2", Port: 5432}

	connector := newMockConnector()
	connector.setResponse(node1.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       node1.String(),
			Hostname: node1.Host,
			Role:     domain.RolePrimary,
			Status:   domain.StatusHealthy,
		},
	})
	connector.setError(node2.String(), errors.New("connection refused"))

	seedProvider := providers.NewStaticSeedProvider(node1, node2)
	collector := topology.NewCollector(
		seedProvider,
		topology.WithConnector(connector),
	)

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph.Nodes))
	}

	foundOffline := false
	for _, n := range graph.Nodes {
		if n.ID == node2.String() && n.Status == domain.StatusOffline {
			foundOffline = true
		}
	}

	if !foundOffline {
		t.Errorf("expected node2 to be marked as OFFLINE")
	}

	if err := graph.Validate(); err != nil {
		t.Errorf("graph validation failed: %v", err)
	}
}

func TestCollector_Collect_CycleProtection(t *testing.T) {
	ctx := context.Background()

	nodeA := domain.TargetEndpoint{Host: "nodeA", Port: 5432}
	nodeB := domain.TargetEndpoint{Host: "nodeB", Port: 5432}

	connector := newMockConnector()
	// nodeA discovers nodeB
	connector.setResponse(nodeA.String(), &topology.NodeInspectionResult{
		Node:            domain.DatabaseNode{ID: nodeA.String(), Hostname: nodeA.Host, Role: domain.RolePrimary, Status: domain.StatusHealthy},
		DiscoveredPeers: []domain.TargetEndpoint{nodeB},
	})
	// nodeB discovers nodeA (цикл)
	connector.setResponse(nodeB.String(), &topology.NodeInspectionResult{
		Node:            domain.DatabaseNode{ID: nodeB.String(), Hostname: nodeB.Host, Role: domain.RoleReplica, Status: domain.StatusHealthy},
		DiscoveredPeers: []domain.TargetEndpoint{nodeA},
	})

	seedProvider := providers.NewStaticSeedProvider(nodeA)
	collector := topology.NewCollector(seedProvider, topology.WithConnector(connector))

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph.Nodes))
	}

	if connector.calls[nodeA.String()] != 1 {
		t.Errorf("expected nodeA to be inspected exactly once, got %d", connector.calls[nodeA.String()])
	}
	if connector.calls[nodeB.String()] != 1 {
		t.Errorf("expected nodeB to be inspected exactly once, got %d", connector.calls[nodeB.String()])
	}
}

func TestCollector_Collect_ContextCanceled(t *testing.T) {
	node := domain.TargetEndpoint{Host: "node1", Port: 5432}
	connector := newMockConnector()

	seedProvider := providers.NewStaticSeedProvider(node)
	collector := topology.NewCollector(seedProvider, topology.WithConnector(connector))

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(canceledCtx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCollector_Collect_SeedErrors(t *testing.T) {
	ctx := context.Background()
	connector := newMockConnector()

	t.Run("empty seeds", func(t *testing.T) {
		emptySeedProvider := providers.NewStaticSeedProvider()
		collector := topology.NewCollector(emptySeedProvider, topology.WithConnector(connector))

		_, err := collector.Collect(ctx)
		if err == nil {
			t.Fatal("expected error on empty seeds, got nil")
		}
	})

	t.Run("nil seed provider", func(t *testing.T) {
		collector := topology.NewCollector(nil, topology.WithConnector(connector))

		_, err := collector.Collect(ctx)
		if err == nil {
			t.Fatal("expected error on nil seed provider, got nil")
		}
	})
}

func TestCollector_Collect_DiscoveryDepth(t *testing.T) {
	ctx := context.Background()

	master := domain.TargetEndpoint{Host: "master", Port: 5432}
	replica1 := domain.TargetEndpoint{Host: "replica1", Port: 5432}
	replica2 := domain.TargetEndpoint{Host: "replica2", Port: 5432} // cascading replica of replica1

	connector := newMockConnector()
	connector.setResponse(master.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       master.String(),
			Hostname: master.Host,
			Role:     domain.RolePrimary,
			Status:   domain.StatusHealthy,
		},
		ReplicationLinks: []domain.ReplicationLink{
			{
				ID:           "repl-master-replica1",
				SourceNodeID: master.String(),
				TargetNodeID: replica1.String(),
				Type:         domain.TypeAsync,
				Status:       domain.ChannelStatusActive,
			},
		},
		DiscoveredPeers: []domain.TargetEndpoint{replica1},
	})

	connector.setResponse(replica1.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       replica1.String(),
			Hostname: replica1.Host,
			Role:     domain.RoleReplica,
			Status:   domain.StatusHealthy,
		},
		ReplicationLinks: []domain.ReplicationLink{
			{
				ID:           "repl-replica1-replica2",
				SourceNodeID: replica1.String(),
				TargetNodeID: replica2.String(),
				Type:         domain.TypeAsync,
				Status:       domain.ChannelStatusActive,
			},
		},
		DiscoveredPeers: []domain.TargetEndpoint{replica2},
	})

	connector.setResponse(replica2.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       replica2.String(),
			Hostname: replica2.Host,
			Role:     domain.RoleCascadingReplica,
			Status:   domain.StatusHealthy,
		},
	})

	seedProvider := providers.NewStaticSeedProvider(master)

	t.Run("depth 0 visits only seeds", func(t *testing.T) {
		collector := topology.NewCollector(
			seedProvider,
			topology.WithConnector(connector),
			topology.WithDiscoveryDepth(0),
		)

		graph, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(graph.Nodes) != 1 {
			t.Fatalf("expected 1 node with depth 0, got %d", len(graph.Nodes))
		}
		if graph.Nodes[0].ID != master.String() {
			t.Errorf("expected master node, got %s", graph.Nodes[0].ID)
		}
	})

	t.Run("depth 1 visits master and direct replica but not cascading", func(t *testing.T) {
		collector := topology.NewCollector(
			seedProvider,
			topology.WithConnector(connector),
			topology.WithDiscoveryDepth(1),
		)

		graph, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(graph.Nodes) != 2 {
			t.Fatalf("expected 2 nodes with depth 1, got %d", len(graph.Nodes))
		}
		if len(graph.Links) != 1 {
			t.Fatalf("expected 1 link with depth 1, got %d", len(graph.Links))
		}
	})

	t.Run("depth 2 visits master, direct replica, and cascading replica", func(t *testing.T) {
		collector := topology.NewCollector(
			seedProvider,
			topology.WithConnector(connector),
			topology.WithDiscoveryDepth(2),
		)

		graph, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(graph.Nodes) != 3 {
			t.Fatalf("expected 3 nodes with depth 2, got %d", len(graph.Nodes))
		}
		if len(graph.Links) != 2 {
			t.Fatalf("expected 2 links with depth 2, got %d", len(graph.Links))
		}
	})
}

func TestCollector_Collect_SplitBrainDetection(t *testing.T) {
	ctx := context.Background()

	nodeA := domain.TargetEndpoint{Host: "pg-primary-a", Port: 5432}
	nodeB := domain.TargetEndpoint{Host: "pg-primary-b", Port: 5432}

	connector := newMockConnector()
	connector.setResponse(nodeA.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       nodeA.String(),
			Hostname: nodeA.Host,
			Role:     domain.RolePrimary,
			Status:   domain.StatusHealthy,
		},
	})
	connector.setResponse(nodeB.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       nodeB.String(),
			Hostname: nodeB.Host,
			Role:     domain.RolePrimary,
			Status:   domain.StatusHealthy,
		},
	})

	seedProvider := providers.NewStaticSeedProvider(nodeA, nodeB)
	collector := topology.NewCollector(
		seedProvider,
		topology.WithConnector(connector),
	)

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !graph.SplitBrainDetected {
		t.Error("expected SplitBrainDetected to be true when two active primaries exist")
	}

	if len(graph.Warnings) == 0 {
		t.Fatal("expected warnings to contain split-brain warning")
	}

	foundWarning := false
	for _, w := range graph.Warnings {
		if strings.Contains(w, "split-brain") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected warning mentioning 'split-brain', got: %v", graph.Warnings)
	}
}

func TestCollector_Collect_ClusterMismatchWarning(t *testing.T) {
	ctx := context.Background()

	nodeA := domain.TargetEndpoint{Host: "node-a", Port: 5432}
	nodeB := domain.TargetEndpoint{Host: "node-b", Port: 5432}

	connector := newMockConnector()
	connector.setResponse(nodeA.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       nodeA.String(),
			Hostname: nodeA.Host,
			Role:     domain.RolePrimary,
			Status:   domain.StatusHealthy,
			Tags:     map[string]string{"system_identifier": "1111111111111"},
		},
	})
	connector.setResponse(nodeB.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       nodeB.String(),
			Hostname: nodeB.Host,
			Role:     domain.RoleReplica,
			Status:   domain.StatusHealthy,
			Tags:     map[string]string{"system_identifier": "2222222222222"},
		},
	})

	seedProvider := providers.NewStaticSeedProvider(nodeA, nodeB)
	collector := topology.NewCollector(
		seedProvider,
		topology.WithConnector(connector),
	)

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundWarning := false
	for _, w := range graph.Warnings {
		if strings.Contains(w, "cluster system identifier mismatch") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected system identifier mismatch warning, got: %v", graph.Warnings)
	}
}

func TestCollector_Collect_NoPrimaryWarning(t *testing.T) {
	ctx := context.Background()

	nodeReplica := domain.TargetEndpoint{Host: "node-replica", Port: 5432}

	connector := newMockConnector()
	connector.setResponse(nodeReplica.String(), &topology.NodeInspectionResult{
		Node: domain.DatabaseNode{
			ID:       nodeReplica.String(),
			Hostname: nodeReplica.Host,
			Role:     domain.RoleReplica,
			Status:   domain.StatusHealthy,
		},
	})

	seedProvider := providers.NewStaticSeedProvider(nodeReplica)
	collector := topology.NewCollector(
		seedProvider,
		topology.WithConnector(connector),
	)

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundWarning := false
	for _, w := range graph.Warnings {
		if strings.Contains(w, "no active primary node detected") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected no primary warning, got: %v", graph.Warnings)
	}
}

func TestCollector_Collect_ParallelExecution(t *testing.T) {
	ctx := context.Background()

	nodes := make([]domain.TargetEndpoint, 4)
	connector := newMockConnector()

	for i := 0; i < 4; i++ {
		nodes[i] = domain.TargetEndpoint{Host: fmt.Sprintf("node-%d", i), Port: 5432}
		connector.setResponse(nodes[i].String(), &topology.NodeInspectionResult{
			Node: domain.DatabaseNode{
				ID:       nodes[i].String(),
				Hostname: nodes[i].Host,
				Role:     domain.RoleReplica,
				Status:   domain.StatusHealthy,
			},
		})
	}

	seedProvider := providers.NewStaticSeedProvider(nodes...)
	collector := topology.NewCollector(
		seedProvider,
		topology.WithConnector(connector),
		topology.WithMaxConcurrency(4),
		topology.WithPerNodeTimeout(1*time.Second),
	)

	graph, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(graph.Nodes))
	}
}
