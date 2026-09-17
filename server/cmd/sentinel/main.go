// Package main is the entry point for the DB-Sentinel core service.
package main

import (
	"DBSentinel/cmd/seeder"
	"DBSentinel/config"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.GetConfig()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.PgMasterHost, cfg.PgMasterUser, cfg.PgMasterPassword, cfg.PgMasterDBName, cfg.PgMasterPort)

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
}
