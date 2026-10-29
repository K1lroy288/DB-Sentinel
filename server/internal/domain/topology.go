package domain

// NodeRole представляет роль узла базы данных в кластере.
type NodeRole string

// HealthStatus представляет статус работоспособности узла базы данных.
type HealthStatus string

// Роли узлов в кластере базы данных.
const (
	RolePrimary          NodeRole = "PRIMARY"
	RoleReplica          NodeRole = "REPLICA"
	RoleCascadingReplica NodeRole = "CASCADING_REPLICA"
	RoleArbiter          NodeRole = "ARBITER"
)

// Статусы работоспособности узла базы данных.
const (
	StatusHealthy  HealthStatus = "HEALTHY"
	StatusLagging  HealthStatus = "LAGGING"
	StatusCritical HealthStatus = "CRITICAL"
	StatusOffline  HealthStatus = "OFFLINE"
)

// NodeMetrics содержит метрики производительности и репликации узла базы данных.
type NodeMetrics struct {
	ReplicationLagBytes   int64   `json:"replicationLagBytes"`
	ReplicationLagSeconds float64 `json:"replicationLagSeconds"`
	TransactionsPerSecond float64 `json:"transactionsPerSecond"`
	CPULoadPercent        float64 `json:"cpuLoadPercent"`
}

// DatabaseNode представляет отдельный экземпляр базы данных в топологии кластера.
type DatabaseNode struct {
	ID       string            `json:"id"`
	Hostname string            `json:"hostname"`
	Role     NodeRole          `json:"role"`
	Status   HealthStatus      `json:"status"`
	Version  string            `json:"version"`
	ReadOnly bool              `json:"readOnly"`
	Metrics  NodeMetrics       `json:"metrics"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// ReplicationType определяет используемый механизм репликации между узлами.
type ReplicationType string

// ChannelStatus представляет состояние канала репликации.
type ChannelStatus string

// Типы репликации, поддерживаемые в кластере.
const (
	TypeAsync    ReplicationType = "ASYNC"
	TypeSemiSync ReplicationType = "SEMI_SYNC"
	TypeSync     ReplicationType = "SYNC"
	TypeLogical  ReplicationType = "LOGICAL"
	TypePhysical ReplicationType = "PHYSICAL"
)

// Статусы канала репликации.
const (
	ChannelStatusActive       ChannelStatus = "ACTIVE"
	ChannelStatusPaused       ChannelStatus = "PAUSED"
	ChannelStatusDisconnected ChannelStatus = "DISCONNECTED"
	ChannelStatusRecovering   ChannelStatus = "RECOVERING"
)

// PositionDetails содержит номера позиций в журнале предзаписи (LSN) репликации.
type PositionDetails struct {
	SourceLsn  string `json:"sourceLsn"`
	AppliedLsn string `json:"appliedLsn"`
	SentLsn    string `json:"sentLsn"`
}

// ReplicationLink определяет направленный поток репликации между двумя узлами.
type ReplicationLink struct {
	ID              string          `json:"id"`
	SourceNodeID    string          `json:"sourceNodeId"`
	TargetNodeID    string          `json:"targetNodeId"`
	Type            ReplicationType `json:"type"`
	Status          ChannelStatus   `json:"status"`
	PositionDetails PositionDetails `json:"positionDetails"`
}

// TopologyGraph представляет полный граф топологии кластера в определённый момент времени.
type TopologyGraph struct {
	ClusterName        string            `json:"clusterName"`
	Timestamp          UnixMillisTime    `json:"timestamp"`
	Nodes              []DatabaseNode    `json:"nodes"`
	Links              []ReplicationLink `json:"links"`
	SplitBrainDetected bool              `json:"splitBrainDetected,omitempty"`
	Warnings           []string          `json:"warnings,omitempty"`
}
