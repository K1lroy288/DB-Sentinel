// Package config manages application configuration setup and environment variable binding.
package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// Config holds the application configuration parameters loaded from environment variables.
type Config struct {
	PgMasterHost     string `mapstructure:"PG_MASTER_HOST"`
	PgMasterPort     int    `mapstructure:"PG_MASTER_PORT"`
	PgMasterUser     string `mapstructure:"PG_MASTER_USER"`
	PgMasterPassword string `mapstructure:"PG_MASTER_PASSWORD"`
	PgMasterDBName   string `mapstructure:"PG_MASTER_DB_NAME"`
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig returns a thread-safe singleton instance of the application Config.
func GetConfig() *Config {
	once.Do(func() {
		instance = loadConfig()
	})
	return instance
}

func loadConfig() *Config {
	v := viper.New()

	v.AutomaticEnv()

	for _, key := range []string{
		"PG_MASTER_HOST",
		"PG_MASTER_PORT",
		"PG_MASTER_USER",
		"PG_MASTER_PASSWORD",
		"PG_MASTER_DB_NAME",
	} {
		_ = v.BindEnv(key)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Printf("config unmarshal error: %v", err)
		return nil
	}

	return &cfg
}
