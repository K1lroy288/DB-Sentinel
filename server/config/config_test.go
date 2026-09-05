package config

import (
	"testing"
)

func TestLoadConfig_FromENV(t *testing.T) {
	t.Setenv("PG_MASTER_HOST", "test-host")
	t.Setenv("PG_MASTER_PORT", "5432")
	t.Setenv("PG_MASTER_USER", "test_user")
	t.Setenv("PG_MASTER_PASSWORD", "dummy_pass")
	t.Setenv("PG_MASTER_DB_NAME", "test_db")

	// 2. Загружаем конфиг
	cfg := loadConfig()

	if cfg == nil {
		t.Fatal("expected config to be loaded, got nil")
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"PgMasterHost", cfg.PgMasterHost, "test-host"},
		{"PgMasterPort", cfg.PgMasterPort, 5432},
		{"PgMasterUser", cfg.PgMasterUser, "test_user"},
		{"PgMasterPassword", cfg.PgMasterPassword, "dummy_pass"},
		{"PgMasterDBName", cfg.PgMasterDBName, "test_db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %s to be %v, got %v", tt.name, tt.expected, tt.got)
			}
		})
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
