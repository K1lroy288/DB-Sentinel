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

	// Параметры сид-провайдера и дискавери топологии
	SeedProviderType    string `mapstructure:"SEED_PROVIDER_TYPE"`
	SeedStaticAddresses string `mapstructure:"SEED_STATIC_ADDRESSES"`
	ConsulAddr          string `mapstructure:"CONSUL_ADDR"`
	ConsulServiceName   string `mapstructure:"CONSUL_SERVICE_NAME"`
	ConsulToken         string `mapstructure:"CONSUL_TOKEN"`
	ConsulDC            string `mapstructure:"CONSUL_DC"`
	DNSService          string `mapstructure:"DNS_SRV_SERVICE"`
	DNSProto            string `mapstructure:"DNS_SRV_PROTO"`
	DNSName             string `mapstructure:"DNS_SRV_NAME"`
	DiscoveryDepth      int    `mapstructure:"DISCOVERY_DEPTH"`
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

	// Значения по умолчанию
	v.SetDefault("DISCOVERY_DEPTH", 3)
	v.SetDefault("SEED_PROVIDER_TYPE", "static")

	for _, key := range []string{
		"POSTGRES_HOST_MASTER",
		"POSTGRES_PORT_MASTER",
		"POSTGRES_USER_MASTER",
		"POSTGRES_PASSWORD_MASTER",
		"POSTGRES_DB_NAME_MASTER",
		"SEED_PROVIDER_TYPE",
		"SEED_STATIC_ADDRESSES",
		"CONSUL_ADDR",
		"CONSUL_SERVICE_NAME",
		"CONSUL_TOKEN",
		"CONSUL_DC",
		"DNS_SRV_SERVICE",
		"DNS_SRV_PROTO",
		"DNS_SRV_NAME",
		"DISCOVERY_DEPTH",
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
