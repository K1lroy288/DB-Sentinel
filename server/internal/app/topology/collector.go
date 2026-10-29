package topology

import (
	"DBSentinel/internal/domain"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultClusterName — имя кластера по умолчанию.
const DefaultClusterName = "postgres-cluster"

// DefaultCollectionTimeout — общий тайм-аут сбора топологии по умолчанию.
const DefaultCollectionTimeout = 15 * time.Second

// DefaultPerNodeTimeout — тайм-аут на опрос одного узла по умолчанию.
const DefaultPerNodeTimeout = 2 * time.Second

// DefaultMaxConcurrency — максимальное число одновременно опрашиваемых узлов.
const DefaultMaxConcurrency = 16

// DefaultDiscoveryDepth — максимальная глубина рекурсивного обнаружения реплик по умолчанию (3 уровня каскада).
const DefaultDiscoveryDepth = 3

// Option определяет функциональную опцию конфигурации Collector.
type Option func(*Collector)

// WithClusterName задаёт имя опрашиваемого кластера.
func WithClusterName(name string) Option {
	return func(c *Collector) {
		if name != "" {
			c.clusterName = name
		}
	}
}

// WithConnector подставляет пользовательскую реализацию NodeConnector (например, мок для тестов).
func WithConnector(connector NodeConnector) Option {
	return func(c *Collector) {
		if connector != nil {
			c.connector = connector
		}
	}
}

// WithTimeout переопределяет общий тайм-аут сбора топологии.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Collector) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithPerNodeTimeout задаёт индивидуальный тайм-аут на опрос отдельного узла.
func WithPerNodeTimeout(timeout time.Duration) Option {
	return func(c *Collector) {
		if timeout > 0 {
			c.perNodeTimeout = timeout
		}
	}
}

// WithMaxConcurrency задаёт максимальное число одновременных параллельных обращений к узлам.
func WithMaxConcurrency(concurrency int) Option {
	return func(c *Collector) {
		if concurrency > 0 {
			c.maxConcurrency = concurrency
		}
	}
}

// WithDiscoveryDepth задаёт максимальную глубину погружения по репликам.
// 0 — только сиды, 1 — сиды + прямые реплики, n — глубина каскада, -1 — без ограничений.
func WithDiscoveryDepth(depth int) Option {
	return func(c *Collector) {
		c.discoveryDepth = depth
	}
}

// Collector осуществляет обнаружение, опрос и построение графа топологии базы данных.
type Collector struct {
	clusterName    string
	seedProvider   domain.SeedProvider
	connector      NodeConnector
	timeout        time.Duration
	perNodeTimeout time.Duration
	maxConcurrency int
	discoveryDepth int
}

// NewCollector создаёт новый экземпляр Collector с переданным провайдером сидов и опциями.
func NewCollector(seedProvider domain.SeedProvider, opts ...Option) *Collector {
	c := &Collector{
		clusterName:    DefaultClusterName,
		seedProvider:   seedProvider,
		timeout:        DefaultCollectionTimeout,
		perNodeTimeout: DefaultPerNodeTimeout,
		maxConcurrency: DefaultMaxConcurrency,
		discoveryDepth: DefaultDiscoveryDepth,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Collect запускает цикл обнаружения: получает сиды, обходит узлы через параллельный BFS и формирует валидный TopologyGraph.
func (c *Collector) Collect(ctx context.Context) (*domain.TopologyGraph, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	collectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	seeds, err := c.resolveSeeds(collectCtx)
	if err != nil {
		return nil, err
	}

	nodesMap, linksMap, err := c.traverse(collectCtx, seeds)
	if err != nil {
		return nil, err
	}

	return c.buildGraph(nodesMap, linksMap)
}

func (c *Collector) resolveSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if c.seedProvider == nil {
		return nil, fmt.Errorf("seed provider is not configured")
	}

	seeds, err := c.seedProvider.GetSeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve seed endpoints: %w", err)
	}

	if len(seeds) == 0 {
		return nil, fmt.Errorf("no seed endpoints available")
	}

	if c.connector == nil {
		return nil, fmt.Errorf("node connector is not configured")
	}

	return seeds, nil
}

func (c *Collector) traverse(
	ctx context.Context,
	seeds []domain.TargetEndpoint,
) (map[string]domain.DatabaseNode, map[string]domain.ReplicationLink, error) {
	visited := make(map[string]bool)
	nodesMap := make(map[string]domain.DatabaseNode)
	linksMap := make(map[string]domain.ReplicationLink)

	currentLevel := seeds
	depth := 0

	sem := make(chan struct{}, c.maxConcurrency)

	for len(currentLevel) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		// Фильтруем узлы текущего уровня, оставляя только непосещенные
		targetsToInspect := make([]domain.TargetEndpoint, 0, len(currentLevel))
		for _, target := range currentLevel {
			key := target.String()
			if !visited[key] {
				visited[key] = true
				targetsToInspect = append(targetsToInspect, target)
			}
		}

		if len(targetsToInspect) == 0 {
			break
		}

		var (
			mu             sync.Mutex
			wg             sync.WaitGroup
			nextLevelPeers []domain.TargetEndpoint
		)

		for _, target := range targetsToInspect {
			wg.Add(1)
			sem <- struct{}{}

			go func(tgt domain.TargetEndpoint) {
				defer func() {
					<-sem
					wg.Done()
				}()

				nodeCtx, cancel := context.WithTimeout(ctx, c.perNodeTimeout)
				defer cancel()

				res, err := c.connector.InspectNode(nodeCtx, tgt)

				mu.Lock()
				defer mu.Unlock()

				key := tgt.String()
				if err != nil {
					nodesMap[key] = domain.DatabaseNode{
						ID:       key,
						Hostname: tgt.Host,
						Role:     domain.RoleReplica,
						Status:   domain.StatusOffline,
					}
					return
				}

				nodesMap[key] = res.Node
				for _, link := range res.ReplicationLinks {
					linksMap[link.ID] = link
				}

				// Если текущая глубина разрешает дальнейший обход, собираем пиров для следующего уровня
				if c.discoveryDepth < 0 || depth < c.discoveryDepth {
					nextLevelPeers = append(nextLevelPeers, res.DiscoveredPeers...)
				}
			}(target)
		}

		wg.Wait()

		currentLevel = nextLevelPeers
		depth++
	}

	return nodesMap, linksMap, nil
}

func (c *Collector) buildGraph(
	nodesMap map[string]domain.DatabaseNode,
	linksMap map[string]domain.ReplicationLink,
) (*domain.TopologyGraph, error) {
	nodes := make([]domain.DatabaseNode, 0, len(nodesMap))
	for _, node := range nodesMap {
		nodes = append(nodes, node)
	}

	links := filterValidLinks(nodesMap, linksMap)
	splitBrain, warnings := detectGraphAnomalies(nodes)

	graph := &domain.TopologyGraph{
		ClusterName:        c.clusterName,
		Timestamp:          domain.NewUnixMillisTime(time.Now()),
		Nodes:              nodes,
		Links:              links,
		SplitBrainDetected: splitBrain,
		Warnings:           warnings,
	}

	if err := graph.Validate(); err != nil {
		return nil, fmt.Errorf("topology validation failed: %w", err)
	}

	return graph, nil
}

func filterValidLinks(
	nodesMap map[string]domain.DatabaseNode,
	linksMap map[string]domain.ReplicationLink,
) []domain.ReplicationLink {
	links := make([]domain.ReplicationLink, 0, len(linksMap))
	for _, link := range linksMap {
		if _, srcExists := nodesMap[link.SourceNodeID]; !srcExists {
			continue
		}
		if _, tgtExists := nodesMap[link.TargetNodeID]; !tgtExists {
			continue
		}
		links = append(links, link)
	}
	return links
}

func detectGraphAnomalies(nodes []domain.DatabaseNode) (bool, []string) {
	var activePrimaries []string
	systemIDs := make(map[string]bool)

	for _, node := range nodes {
		if node.Status == domain.StatusOffline {
			continue
		}
		if node.Role == domain.RolePrimary {
			activePrimaries = append(activePrimaries, node.ID)
		}
		if sysID, ok := node.Tags["system_identifier"]; ok && sysID != "" {
			systemIDs[sysID] = true
		}
	}

	var warnings []string
	splitBrain := false

	if len(activePrimaries) > 1 {
		splitBrain = true
		warnings = append(warnings, fmt.Sprintf("split-brain detected: multiple active primary nodes found (%s)", strings.Join(activePrimaries, ", ")))
	} else if len(activePrimaries) == 0 && len(nodes) > 0 {
		warnings = append(warnings, "no active primary node detected in cluster (failover or read-only mode)")
	}

	if len(systemIDs) > 1 {
		ids := make([]string, 0, len(systemIDs))
		for id := range systemIDs {
			ids = append(ids, id)
		}
		warnings = append(warnings, fmt.Sprintf("cluster system identifier mismatch: nodes belong to different clusters (%s)", strings.Join(ids, ", ")))
	}

	return splitBrain, warnings
}
