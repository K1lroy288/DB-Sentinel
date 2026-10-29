// Package main является точкой входа для основного сервиса DB-Sentinel.
package main

import (
	"DBSentinel/cmd/seeder"
	"DBSentinel/config"
	"DBSentinel/internal/app/providers"
	"DBSentinel/internal/app/topology"
	internalHttp "DBSentinel/internal/transport/http"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed dist/*
var embeddedDist embed.FS

func main() {
	cfg := config.GetConfig()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.PgHostMaster, cfg.PgUserMaster, cfg.PgPasswordMaster, cfg.PgDBNameMaster, cfg.PgPortMaster)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if seeder.IsDatabaseEmpty(db) {
		log.Println("Database is empty or missing tables. Running seeder...")
		seeder.DBSeeder(db)
	} else {
		log.Println("Database already contains data. Skipping seeder.")
	}

	// 1. Инициализация SeedResolver на основе конфигурации приложения
	staticAddrs := []string{fmt.Sprintf("%s:%d", cfg.PgHostMaster, cfg.PgPortMaster)}
	if cfg.SeedStaticAddresses != "" {
		staticAddrs = strings.Split(cfg.SeedStaticAddresses, ",")
	}

	seedResolver, err := providers.NewSeedResolver(providers.ResolverConfig{
		ProviderType:    providers.ProviderType(cfg.SeedProviderType),
		StaticAddresses: staticAddrs,
		SingleAddress:   fmt.Sprintf("%s:%d", cfg.PgHostMaster, cfg.PgPortMaster),
		ConsulAddr:      cfg.ConsulAddr,
		ConsulService:   cfg.ConsulServiceName,
		ConsulToken:     cfg.ConsulToken,
		ConsulDC:        cfg.ConsulDC,
		DNSService:      cfg.DNSService,
		DNSProto:        cfg.DNSProto,
		DNSName:         cfg.DNSName,
		DefaultPort:     cfg.PgPortMaster,
	})
	if err != nil {
		log.Fatalf("Failed to initialize seed resolver: %v", err)
	}

	// 2. Инициализация коннектора узлов PostgreSQL
	connector := topology.NewPGConnector(topology.ConnectionConfig{
		User:     cfg.PgUserMaster,
		Password: cfg.PgPasswordMaster,
		Database: cfg.PgDBNameMaster,
		SSLMode:  "disable",
	})

	// 3. Инициализация Collector топологии
	collector := topology.NewCollector(
		seedResolver.Provider(),
		topology.WithConnector(connector),
		topology.WithClusterName("postgres-cluster"),
		topology.WithDiscoveryDepth(cfg.DiscoveryDepth),
	)

	// 4. Инициализация встроенного веб-сервера (SPA + WebSocket)
	hub := internalHttp.NewHub()
	staticFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		log.Printf("Warning: failed to initialize embedded frontend: %v", err)
	}

	httpServer := internalHttp.NewServer(internalHttp.Config{Addr: ":8080"}, staticFS, hub)
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	// 5. Первичный сбор топологии
	runCollection(collector, hub)

	// 6. Запуск периодического мониторинга кластера
	log.Println("DB-Sentinel daemon started. Web UI available on http://localhost:8080")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			log.Println("Shutting down DB-Sentinel daemon...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(shutdownCtx)
			return
		case <-ticker.C:
			runCollection(collector, hub)
		}
	}
}

func runCollection(collector *topology.Collector, hub *internalHttp.Hub) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topo, err := collector.Collect(ctx)
	if err != nil {
		log.Printf("Topology collection warning: %v", err)
		return
	}

	// Транслируем обновлённую топологию всем клиентам через WebSocket
	if hub != nil {
		hub.Broadcast(topo)
	}

	log.Printf("Topology collected: cluster=%s, nodes=%d, links=%d",
		topo.ClusterName, len(topo.Nodes), len(topo.Links))
	if topo.SplitBrainDetected {
		log.Printf(" [CRITICAL ALERT] SPLIT-BRAIN DETECTED in cluster %s!", topo.ClusterName)
	}
	for _, w := range topo.Warnings {
		log.Printf(" [CLUSTER WARNING] %s", w)
	}
	for _, n := range topo.Nodes {
		log.Printf(" -> Node %s: role=%s, status=%s, version=%s, lagBytes=%d, lagSec=%.2f",
			n.ID, n.Role, n.Status, n.Version, n.Metrics.ReplicationLagBytes, n.Metrics.ReplicationLagSeconds)
	}
	for _, l := range topo.Links {
		log.Printf(" -> Link %s: %s -> %s (type=%s, status=%s)", l.ID, l.SourceNodeID, l.TargetNodeID, l.Type, l.Status)
	}
}
