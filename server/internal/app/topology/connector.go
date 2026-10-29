// Package topology предоставляет примитивы и сервисы для обнаружения и сбора топологии кластера баз данных.
package topology

import (
	"DBSentinel/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"time"

	// Регистрируем драйвер pgx для использования через database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DefaultConnectTimeout определяет тайм-аут подключения к узлу базы данных по умолчанию.
const DefaultConnectTimeout = 5 * time.Second

// NodeInspectionResult содержит результаты инспекции отдельного узла базы данных.
type NodeInspectionResult struct {
	Node             domain.DatabaseNode
	ReplicationLinks []domain.ReplicationLink
	DiscoveredPeers  []domain.TargetEndpoint
}

// NodeConnector абстрагирует подключение к узлу базы данных и сбор его локального состояния.
type NodeConnector interface {
	InspectNode(ctx context.Context, target domain.TargetEndpoint) (*NodeInspectionResult, error)
}

// ConnectionConfig содержит учётные данные и параметры подключения к узлам PostgreSQL.
type ConnectionConfig struct {
	User           string
	Password       string
	Database       string
	SSLMode        string
	ConnectTimeout time.Duration
}

// PGConnector реализует интерфейс NodeConnector для PostgreSQL.
type PGConnector struct {
	config ConnectionConfig
}

// NewPGConnector создаёт новый экземпляр PGConnector с переданной конфигурацией подключения.
func NewPGConnector(cfg ConnectionConfig) *PGConnector {
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = DefaultConnectTimeout
	}
	if cfg.Database == "" {
		cfg.Database = "postgres"
	}

	return &PGConnector{
		config: cfg,
	}
}

// InspectNode подключается к целевому узлу PostgreSQL и выполняет его инспекцию.
func (c *PGConnector) InspectNode(ctx context.Context, target domain.TargetEndpoint) (*NodeInspectionResult, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		c.config.User,
		c.config.Password,
		target.Host,
		target.Port,
		c.config.Database,
		c.config.SSLMode,
		int(c.config.ConnectTimeout.Seconds()),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to %s: %w", target.String(), err)
	}
	defer func() {
		_ = db.Close()
	}()

	connCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(connCtx); err != nil {
		return nil, fmt.Errorf("ping failed for %s: %w", target.String(), err)
	}

	return inspectPostgresNode(ctx, db, target)
}
