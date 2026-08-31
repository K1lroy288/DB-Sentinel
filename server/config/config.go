package config

import (
	"errors"
	"log"
	"strings"
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
	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			log.Printf("warning: .env file is not found, read from ENV: %v", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("config unmarshal error: %v", err)
	}

	instance = &cfg
}
