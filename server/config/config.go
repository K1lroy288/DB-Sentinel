package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

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

func GetConfig() *Config {
	once.Do(loadConfig)
	return instance
}

func loadConfig() {
	viper.AddConfigPath("../cmd")
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf(".env file read error: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("config unmarshal error: %v", err)
	}

	instance = &cfg
}
