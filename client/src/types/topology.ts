export type NodeRole = 'PRIMARY' | 'REPLICA' | 'CASCADING_REPLICA' | 'ARBITER';

export type HealthStatus = 'HEALTHY' | 'LAGGING' | 'CRITICAL' | 'OFFLINE';

export type ReplicationType = 'ASYNC' | 'SEMI_SYNC' | 'SYNC' | 'LOGICAL' | 'PHYSICAL';

export type ChannelStatus = 'ACTIVE' | 'PAUSED' | 'DISCONNECTED' | 'RECOVERING';

export interface NodeMetrics {
  replicationLagBytes: number;
  replicationLagSeconds: number;
  transactionsPerSecond: number;
  cpuLoadPercent: number;
}

export interface DatabaseNode {
  id: string;
  hostname: string;
  role: NodeRole;
  status: HealthStatus;
  version: string;
  readOnly: boolean;
  metrics: NodeMetrics;
  tags?: Record<string, string>;
}

export interface PositionDetails {
  sourceLsn: string;
  appliedLsn: string;
  sentLsn: string;
}

export interface ReplicationLink {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  type: ReplicationType;
  status: ChannelStatus;
  positionDetails: PositionDetails;
}

export interface TopologyGraph {
  clusterName: string;
  timestamp: number;
  nodes: DatabaseNode[];
  links: ReplicationLink[];
  splitBrainDetected?: boolean;
  warnings?: string[];
}

