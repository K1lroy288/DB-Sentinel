// Package config управляет настройкой конфигурации приложения и привязкой переменных окружения.
package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// Config содержит параметры конфигурации приложения, загруженные из переменных окружения.
type Config struct {
	PgHostMaster     string `mapstructure:"POSTGRES_HOST_MASTER"`
	PgPortMaster     int    `mapstructure:"POSTGRES_PORT_MASTER"`
	PgUserMaster     string `mapstructure:"POSTGRES_USER_MASTER"`
	PgPasswordMaster string `mapstructure:"POSTGRES_PASSWORD_MASTER"`
	PgDBNameMaster   string `mapstructure:"POSTGRES_DB_NAME_MASTER"`
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig возвращает потокобезопасный синглтон-экземпляр конфигурации приложения Config.
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
		"POSTGRES_HOST_MASTER",
		"POSTGRES_PORT_MASTER",
		"POSTGRES_USER_MASTER",
		"POSTGRES_PASSWORD_MASTER",
		"POSTGRES_DB_NAME_MASTER",
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
