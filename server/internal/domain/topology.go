package domain

type NodeRole string
type HealthStatus string

const (
	RolePrimary          NodeRole = "PRIMARY"
	RoleReplica          NodeRole = "REPLICA"
	RoleCascadingReplica NodeRole = "CASCADING_REPLICA"
	RoleArbiter          NodeRole = "ARBITER"

	StatusHealthy  HealthStatus = "HEALTHY"
	StatusLagging  HealthStatus = "LAGGING"
	StatusCritical HealthStatus = "CRITICAL"
	StatusOffline  HealthStatus = "OFFLINE"
)

type NodeMetrics struct {
	ReplicationLagBytes   int64   `json:"replicationLagBytes"`
	ReplicationLagSeconds float64 `json:"replicationLagSeconds"`
	TransactionsPerSecond float64 `json:"transactionsPerSecond"`
	CPULoadPercent        float64 `json:"cpuLoadPercent"`
}

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

type ReplicationType string
type ChannelStatus string

const (
	TypeAsync    ReplicationType = "ASYNC"
	TypeSemiSync ReplicationType = "SEMI_SYNC"
	TypeSync     ReplicationType = "SYNC"
	TypeLogical  ReplicationType = "LOGICAL"
	TypePhysical ReplicationType = "PHYSICAL"

	ChannelStatusActive       ChannelStatus = "ACTIVE"
	ChannelStatusPaused       ChannelStatus = "PAUSED"
	ChannelStatusDisconnected ChannelStatus = "DISCONNECTED"
	ChannelStatusRecovering   ChannelStatus = "RECOVERING"
)

type PositionDetails struct {
	SourceLsn  string `json:"sourceLsn"`
	AppliedLsn string `json:"appliedLsn"`
	SentLsn    string `json:"sentLsn"`
}

type ReplicationLink struct {
	ID              string          `json:"id"`
	SourceNodeID    string          `json:"sourceNodeId"`
	TargetNodeID    string          `json:"targetNodeId"`
	Type            ReplicationType `json:"type"`
	Status          ChannelStatus   `json:"status"`
	PositionDetails PositionDetails `json:"positionDetails"`
}

type TopologyGraph struct {
	ClusterName string            `json:"clusterName"`
	Timestamp   UnixMillisTime    `json:"timestamp"`
	Nodes       []DatabaseNode    `json:"nodes"`
	Links       []ReplicationLink `json:"links"`
}
