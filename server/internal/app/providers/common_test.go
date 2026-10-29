package providers_test

import (
	"DBSentinel/internal/app/providers"
	"errors"
	"testing"
)

const (
	testPgMaster  = "postgres-master"
	testPgReplica = "postgres-replica"
	testIPAddr    = "10.0.0.1:5432"
	testHost10    = "10.0.0.1"
)

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		defaultPort int
		wantHost    string
		wantPort    int
		wantErr     error
	}{
		{
			name:        "host only with default port",
			raw:         testPgMaster,
			defaultPort: 5432,
			wantHost:    testPgMaster,
			wantPort:    5432,
			wantErr:     nil,
		},
		{
			name:        "host and port",
			raw:         "postgres-replica:5433",
			defaultPort: 5432,
			wantHost:    testPgReplica,
			wantPort:    5433,
			wantErr:     nil,
		},
		{
			name:        "ipv4 address with port",
			raw:         "192.168.1.100:5432",
			defaultPort: 5432,
			wantHost:    "192.168.1.100",
			wantPort:    5432,
			wantErr:     nil,
		},
		{
			name:        "ipv6 bracketed with port",
			raw:         "[2001:db8::1]:5432",
			defaultPort: 5432,
			wantHost:    "2001:db8::1",
			wantPort:    5432,
			wantErr:     nil,
		},
		{
			name:        "ipv6 bracketed without port",
			raw:         "[::1]",
			defaultPort: 5432,
			wantHost:    "::1",
			wantPort:    5432,
			wantErr:     nil,
		},
		{
			name:        "quoted raw address",
			raw:         `"10.0.0.1:5432"`,
			defaultPort: 5432,
			wantHost:    testHost10,
			wantPort:    5432,
			wantErr:     nil,
		},
		{
			name:        "empty address",
			raw:         "   ",
			defaultPort: 5432,
			wantErr:     providers.ErrEmptyHostAddress,
		},
		{
			name:        "invalid port string",
			raw:         "db.example.com:abc",
			defaultPort: 5432,
			wantErr:     providers.ErrInvalidPortRange,
		},
		{
			name:        "port out of range high",
			raw:         "db.example.com:70000",
			defaultPort: 5432,
			wantErr:     providers.ErrInvalidPortRange,
		},
		{
			name:        "port zero",
			raw:         "db.example.com:0",
			defaultPort: 5432,
			wantErr:     providers.ErrInvalidPortRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, err := providers.ParseEndpoint(tt.raw, tt.defaultPort)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if endpoint.Host != tt.wantHost {
				t.Errorf("expected host %q, got %q", tt.wantHost, endpoint.Host)
			}
			if endpoint.Port != tt.wantPort {
				t.Errorf("expected port %d, got %d", tt.wantPort, endpoint.Port)
			}
		})
	}
}

func TestNormalizeAndDeduplicate(t *testing.T) {
	t.Run("deduplicates identical and comma-separated addresses", func(t *testing.T) {
		raw := []string{
			testIPAddr,
			testIPAddr,
			"10.0.0.2:5432, 10.0.0.3:5433",
			"10.0.0.2:5432",
			"  ",
		}

		endpoints, err := providers.NormalizeAndDeduplicate(raw, 5432)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(endpoints) != 3 {
			t.Fatalf("expected 3 unique endpoints, got %d", len(endpoints))
		}

		expected := []string{
			testIPAddr,
			"10.0.0.2:5432",
			"10.0.0.3:5433",
		}

		for i, exp := range expected {
			if endpoints[i].String() != exp {
				t.Errorf("seed[%d]: expected %s, got %s", i, exp, endpoints[i].String())
			}
		}
	})

	t.Run("returns ErrNoSeedsFound for empty slice", func(t *testing.T) {
		_, err := providers.NormalizeAndDeduplicate([]string{" ", ""}, 5432)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})

	t.Run("returns error on invalid endpoint in list", func(t *testing.T) {
		_, err := providers.NormalizeAndDeduplicate([]string{"10.0.0.1:5432", "invalid-port:999999"}, 5432)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
