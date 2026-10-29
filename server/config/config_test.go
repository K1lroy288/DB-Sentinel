package config

import (
	"testing"
)

func TestLoadConfig_FromENV(t *testing.T) {
	t.Setenv("POSTGRES_HOST_MASTER", "test-host")
	t.Setenv("POSTGRES_PORT_MASTER", "5432")
	t.Setenv("POSTGRES_USER_MASTER", "test_user")
	t.Setenv("POSTGRES_PASSWORD_MASTER", "dummy_pass")
	t.Setenv("POSTGRES_DB_NAME_MASTER", "test_db")
	t.Setenv("SEED_PROVIDER_TYPE", "fallback")
	t.Setenv("SEED_STATIC_ADDRESSES", "node1:5432,node2:5432")
	t.Setenv("CONSUL_ADDR", "http://127.0.0.1:8500")
	t.Setenv("CONSUL_SERVICE_NAME", "postgres-service")
	t.Setenv("DISCOVERY_DEPTH", "2")

	cfg := loadConfig()

	if cfg == nil {
		t.Fatal("expected config to be loaded, got nil")
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"PgMasterHost", cfg.PgHostMaster, "test-host"},
		{"PgMasterPort", cfg.PgPortMaster, 5432},
		{"PgMasterUser", cfg.PgUserMaster, "test_user"},
		{"PgMasterPassword", cfg.PgPasswordMaster, "dummy_pass"},
		{"PgMasterDBName", cfg.PgDBNameMaster, "test_db"},
		{"SeedProviderType", cfg.SeedProviderType, "fallback"},
		{"SeedStaticAddresses", cfg.SeedStaticAddresses, "node1:5432,node2:5432"},
		{"ConsulAddr", cfg.ConsulAddr, "http://127.0.0.1:8500"},
		{"ConsulServiceName", cfg.ConsulServiceName, "postgres-service"},
		{"DiscoveryDepth", cfg.DiscoveryDepth, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %s to be %v, got %v", tt.name, tt.expected, tt.got)
			}
		})
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := loadConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.DiscoveryDepth <= 0 {
		t.Errorf("expected default DiscoveryDepth > 0, got %d", cfg.DiscoveryDepth)
	}
}

func TestGetConfig_Singleton(t *testing.T) {
	cfg1 := GetConfig()
	cfg2 := GetConfig()

	if cfg1 == nil || cfg2 == nil {
		t.Fatal("expected non-nil config instances")
	}

	if cfg1 != cfg2 {
		t.Errorf("expected singleton pointers to be equal, got %p and %p", cfg1, cfg2)
	}
}
