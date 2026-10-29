// Package main является точкой входа для основного сервиса DB-Sentinel.
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
}
