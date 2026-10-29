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
		{"PgMasterHost", cfg.PgHostMaster, "test-host"},
		{"PgMasterPort", cfg.PgPortMaster, 5432},
		{"PgMasterUser", cfg.PgUserMaster, "test_user"},
		{"PgMasterPassword", cfg.PgPasswordMaster, "dummy_pass"},
		{"PgMasterDBName", cfg.PgDBNameMaster, "test_db"},
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
