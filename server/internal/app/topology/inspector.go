package topology

import (
	"DBSentinel/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// inspectPostgresNode выполняет системные запросы к подключению PostgreSQL и извлекает топологические свойства.
func inspectPostgresNode(ctx context.Context, db *sql.DB, target domain.TargetEndpoint) (*NodeInspectionResult, error) {
	var inRecovery bool
	if err := db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return nil, fmt.Errorf("failed to check recovery state: %w", err)
	}

	var version string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		version = "PostgreSQL"
	}

	tags := extractSystemTags(ctx, db)

	node := domain.DatabaseNode{
		ID:       target.String(),
		Hostname: target.Host,
		Role:     domain.RolePrimary,
		Status:   domain.StatusHealthy,
		Version:  cleanVersion(version),
		ReadOnly: false,
		Metrics:  domain.NodeMetrics{},
		Tags:     tags,
	}

	if !inRecovery {
		links, peers, _ := queryPrimaryReplication(ctx, db, target)
		return &NodeInspectionResult{
			Node:             node,
			ReplicationLinks: links,
			DiscoveredPeers:  peers,
		}, nil
	}

	node.Role = domain.RoleReplica
	node.ReadOnly = true

	return inspectStandbyNode(ctx, db, target, node)
}

func extractSystemTags(ctx context.Context, db *sql.DB) map[string]string {
	tags := make(map[string]string)
	var systemID string
	if err := db.QueryRowContext(ctx, "SELECT system_identifier::text FROM pg_control_system()").Scan(&systemID); err == nil && systemID != "" {
		tags["system_identifier"] = systemID
	}
	return tags
}

func inspectStandbyNode(
	ctx context.Context,
	db *sql.DB,
	target domain.TargetEndpoint,
	node domain.DatabaseNode,
) (*NodeInspectionResult, error) {
	var links []domain.ReplicationLink
	var peers []domain.TargetEndpoint

	replicaLinks, discoveredMaster, lagBytes, lagSeconds, err := queryReplicaReceiver(ctx, db, target)
	if err == nil {
		node.Metrics.ReplicationLagBytes = lagBytes
		node.Metrics.ReplicationLagSeconds = lagSeconds
		node.Status = calculateReplicaHealth(lagBytes, lagSeconds)

		links = append(links, replicaLinks...)
		if discoveredMaster != nil {
			peers = append(peers, *discoveredMaster)
		}
	}

	downstreamLinks, discoveredChildren, err := queryPrimaryReplication(ctx, db, target)
	if err == nil && len(downstreamLinks) > 0 {
		node.Role = domain.RoleCascadingReplica
		links = append(links, downstreamLinks...)
		peers = append(peers, discoveredChildren...)
	}

	return &NodeInspectionResult{
		Node:             node,
		ReplicationLinks: links,
		DiscoveredPeers:  peers,
	}, nil
}

func calculateReplicaHealth(lagBytes int64, lagSeconds float64) domain.HealthStatus {
	if lagBytes > 1024*1024*1024 || lagSeconds > 300 {
		return domain.StatusCritical
	}
	if lagBytes > 100*1024*1024 || lagSeconds > 30 {
		return domain.StatusLagging
	}
	return domain.StatusHealthy
}

// queryPrimaryReplication извлекает список подключённых реплик из pg_stat_replication.
func queryPrimaryReplication(
	ctx context.Context,
	db *sql.DB,
	masterTarget domain.TargetEndpoint,
) ([]domain.ReplicationLink, []domain.TargetEndpoint, error) {
	query := `
		SELECT 
			COALESCE(host(client_addr), client_addr::text, ''),
			COALESCE(state, ''),
			COALESCE(sync_state, ''),
			COALESCE(sent_lsn::text, ''),
			COALESCE(replay_lsn::text, '')
		FROM pg_stat_replication
		WHERE client_addr IS NOT NULL;
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var links []domain.ReplicationLink
	var peers []domain.TargetEndpoint

	for rows.Next() {
		var clientAddr, state, syncState, sentLsn, replayLsn string
		if err := rows.Scan(&clientAddr, &state, &syncState, &sentLsn, &replayLsn); err != nil {
			continue
		}

		clientAddr = strings.TrimSpace(clientAddr)
		if idx := strings.Index(clientAddr, "/"); idx != -1 {
			clientAddr = clientAddr[:idx]
		}
		if clientAddr == "" {
			continue
		}

		peerEndpoint := domain.TargetEndpoint{
			Host: clientAddr,
			Port: masterTarget.Port,
		}
		peers = append(peers, peerEndpoint)

		channelStatus := domain.ChannelStatusActive
		if state != "streaming" {
			channelStatus = domain.ChannelStatusRecovering
		}

		replType := domain.TypeAsync
		if syncState == "sync" {
			replType = domain.TypeSync
		}

		link := domain.ReplicationLink{
			ID:           fmt.Sprintf("repl-%s-%s", masterTarget.String(), peerEndpoint.String()),
			SourceNodeID: masterTarget.String(),
			TargetNodeID: peerEndpoint.String(),
			Type:         replType,
			Status:       channelStatus,
			PositionDetails: domain.PositionDetails{
				SourceLsn:  sentLsn,
				AppliedLsn: replayLsn,
				SentLsn:    sentLsn,
			},
		}
		links = append(links, link)
	}

	return links, peers, rows.Err()
}

// queryReplicaReceiver считывает состояние WAL receiver на реплике.
func queryReplicaReceiver(
	ctx context.Context,
	db *sql.DB,
	replicaTarget domain.TargetEndpoint,
) ([]domain.ReplicationLink, *domain.TargetEndpoint, int64, float64, error) {
	query := `
		SELECT 
			COALESCE(status, ''),
			COALESCE(receive_start_lsn::text, ''),
			COALESCE(received_lsn::text, ''),
			COALESCE(latest_end_lsn::text, ''),
			COALESCE(sender_host, ''),
			COALESCE(sender_port, 0),
			COALESCE(pg_wal_lsn_diff(pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn()), 0),
			COALESCE(EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())), 0),
			COALESCE(EXTRACT(EPOCH FROM (now() - last_msg_receipt_time)), 0)
		FROM pg_stat_wal_receiver;
	`

	var status, startLsn, receivedLsn, latestEndLsn, senderHost string
	var senderPort int
	var lagBytes int64
	var xactLagSeconds, msgLagSeconds float64

	row := db.QueryRowContext(ctx, query)
	err := row.Scan(&status, &startLsn, &receivedLsn, &latestEndLsn, &senderHost, &senderPort, &lagBytes, &xactLagSeconds, &msgLagSeconds)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, 0, 0, nil
		}
		return nil, nil, 0, 0, err
	}

	var lagSeconds float64
	if lagBytes == 0 {
		lagSeconds = 0
	} else if xactLagSeconds > 0 {
		lagSeconds = xactLagSeconds
	} else if msgLagSeconds > 0 {
		lagSeconds = msgLagSeconds
	}

	senderHost = strings.TrimSpace(senderHost)
	if idx := strings.Index(senderHost, "/"); idx != -1 {
		senderHost = senderHost[:idx]
	}
	if senderHost == "" {
		return nil, nil, lagBytes, lagSeconds, nil
	}

	if senderPort <= 0 {
		senderPort = replicaTarget.Port
	}

	masterEndpoint := domain.TargetEndpoint{
		Host: senderHost,
		Port: senderPort,
	}

	channelStatus := domain.ChannelStatusActive
	if status != "streaming" {
		channelStatus = domain.ChannelStatusRecovering
	}

	link := domain.ReplicationLink{
		ID:           fmt.Sprintf("repl-%s-%s", masterEndpoint.String(), replicaTarget.String()),
		SourceNodeID: masterEndpoint.String(),
		TargetNodeID: replicaTarget.String(),
		Type:         domain.TypePhysical,
		Status:       channelStatus,
		PositionDetails: domain.PositionDetails{
			SourceLsn:  startLsn,
			AppliedLsn: receivedLsn,
			SentLsn:    latestEndLsn,
		},
	}

	return []domain.ReplicationLink{link}, &masterEndpoint, lagBytes, lagSeconds, nil
}

// cleanVersion обрезает подробности сборки из строки версии PostgreSQL (например, "PostgreSQL 18.0 on x86_64...").
func cleanVersion(v string) string {
	parts := strings.Split(v, " on ")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return v
}
