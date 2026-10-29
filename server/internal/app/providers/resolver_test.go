package providers_test

import (
	"DBSentinel/internal/app/providers"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testResolverConsulHost = "10.0.0.1"

func TestNewSeedResolver_Static(t *testing.T) {
	ctx := context.Background()

	t.Run("valid static addresses", func(t *testing.T) {
		resolver, err := providers.NewSeedResolver(providers.ResolverConfig{
			ProviderType:    providers.ProviderTypeStatic,
			StaticAddresses: []string{"master:5432", "replica:5433"},
			DefaultPort:     5432,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds, err := resolver.ResolveSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected resolve error: %v", err)
		}

		if len(seeds) != 2 {
			t.Fatalf("expected 2 seeds, got %d", len(seeds))
		}
		if resolver.Provider() == nil {
			t.Errorf("expected non-nil provider")
		}
	})

	t.Run("empty static addresses returns error", func(t *testing.T) {
		_, err := providers.NewSeedResolver(providers.ResolverConfig{
			ProviderType:    providers.ProviderTypeStatic,
			StaticAddresses: nil,
		})
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})
}

func TestNewSeedResolver_Single(t *testing.T) {
	ctx := context.Background()

	t.Run("valid single entrypoint", func(t *testing.T) {
		resolver, err := providers.NewSeedResolver(providers.ResolverConfig{
			ProviderType:  providers.ProviderTypeSingle,
			SingleAddress: "vip.cluster.local:5432",
			DefaultPort:   5432,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds, err := resolver.ResolveSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected resolve error: %v", err)
		}

		if len(seeds) != 1 || seeds[0].Host != "vip.cluster.local" {
			t.Fatalf("expected single seed, got %+v", seeds)
		}
	})

	t.Run("multiple addresses in single returns error", func(t *testing.T) {
		_, err := providers.NewSeedResolver(providers.ResolverConfig{
			ProviderType:  providers.ProviderTypeSingle,
			SingleAddress: "node1:5432, node2:5432",
		})
		if !errors.Is(err, providers.ErrSingleEntrypointMultiple) {
			t.Fatalf("expected ErrSingleEntrypointMultiple, got %v", err)
		}
	})
}

func TestNewSeedResolver_Env(t *testing.T) {
	ctx := context.Background()
	envKey := "TEST_RESOLVER_ENV_VAR"
	t.Setenv(envKey, "env-node:5432")

	resolver, err := providers.NewSeedResolver(providers.ResolverConfig{
		ProviderType: providers.ProviderTypeEnv,
		EnvVarName:   envKey,
		DefaultPort:  5432,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seeds, err := resolver.ResolveSeeds(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(seeds) != 1 || seeds[0].Host != "env-node" {
		t.Fatalf("expected env-node, got %+v", seeds)
	}
}

func TestNewSeedResolver_Consul(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"Node": {"Address": "192.168.1.10"}, "Service": {"Address": "10.0.0.1", "Port": 5432}}]`))
	}))
	defer ts.Close()

	resolver, err := providers.NewSeedResolver(providers.ResolverConfig{
		ProviderType:  providers.ProviderTypeConsul,
		ConsulAddr:    ts.URL,
		ConsulService: "pg-cluster",
		ConsulClient:  ts.Client(),
		DefaultPort:   5432,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seeds, err := resolver.ResolveSeeds(ctx)
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}

	if len(seeds) != 1 || seeds[0].Host != testResolverConsulHost {
		t.Fatalf("expected %s, got %+v", testResolverConsulHost, seeds)
	}
}

func TestNewSeedResolver_DNS(t *testing.T) {
	ctx := context.Background()

	mock := &mockSRVResolver{
		addrs: []*net.SRV{{Target: "dns-node.internal.", Port: 5432}},
	}

	resolver, err := providers.NewSeedResolver(providers.ResolverConfig{
		ProviderType: providers.ProviderTypeDNS,
		DNSName:      "pg.service.consul",
		DNSResolver:  mock,
		DefaultPort:  5432,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seeds, err := resolver.ResolveSeeds(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(seeds) != 1 || seeds[0].Host != "dns-node.internal" {
		t.Fatalf("expected dns-node.internal, got %+v", seeds)
	}
}

func TestNewSeedResolver_Fallback(t *testing.T) {
	ctx := context.Background()

	// Consul возвращает 500 ошибку, цепочка должна откатиться на static
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	resolver, err := providers.NewSeedResolver(providers.ResolverConfig{
		ProviderType:    providers.ProviderTypeFallback,
		ConsulAddr:      ts.URL,
		ConsulService:   "failing-service",
		ConsulClient:    ts.Client(),
		StaticAddresses: []string{"fallback-static:5432"},
		DefaultPort:     5432,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seeds, err := resolver.ResolveSeeds(ctx)
	if err != nil {
		t.Fatalf("unexpected fallback error: %v", err)
	}

	if len(seeds) != 1 || seeds[0].Host != "fallback-static" {
		t.Fatalf("expected fallback-static, got %+v", seeds)
	}
}

func TestNewSeedResolver_Invalid(t *testing.T) {
	_, err := providers.NewSeedResolver(providers.ResolverConfig{
		ProviderType: "unknown-type",
	})
	if err == nil {
		t.Fatal("expected error on unknown provider type, got nil")
	}
}
