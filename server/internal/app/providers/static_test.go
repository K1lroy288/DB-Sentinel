package providers_test

import (
	"DBSentinel/internal/app/providers"
	"DBSentinel/internal/domain"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

const (
	testStaticMaster  = "postgres-master"
	testStaticReplica = "postgres-replica"
	testStaticHost1   = "host1"
)

func TestStaticSeedProvider_GetSeeds(t *testing.T) {
	ctx := context.Background()

	t.Run("valid endpoints", func(t *testing.T) {
		p := providers.NewStaticSeedProvider(
			domain.TargetEndpoint{Host: testStaticMaster, Port: 5432},
			domain.TargetEndpoint{Host: testStaticReplica, Port: 5433},
			domain.TargetEndpoint{Host: testStaticMaster, Port: 5432}, // дубликат
		)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(seeds) != 2 {
			t.Fatalf("expected 2 unique seeds, got %d", len(seeds))
		}
		if seeds[0].Host != testStaticMaster || seeds[0].Port != 5432 {
			t.Errorf("unexpected seed[0]: %+v", seeds[0])
		}
		if seeds[1].Host != testStaticReplica || seeds[1].Port != 5433 {
			t.Errorf("unexpected seed[1]: %+v", seeds[1])
		}
	})

	t.Run("empty provider returns ErrNoSeedsFound", func(t *testing.T) {
		p := providers.NewStaticSeedProvider()
		seeds, err := p.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
		if seeds != nil {
			t.Errorf("expected nil seeds, got %+v", seeds)
		}
	})

	t.Run("defensive copy prevents external mutation", func(t *testing.T) {
		p := providers.NewStaticSeedProvider(
			domain.TargetEndpoint{Host: testStaticHost1, Port: 5432},
		)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds[0].Host = "mutated-host"

		seedsAgain, _ := p.GetSeeds(ctx)
		if seedsAgain[0].Host == "mutated-host" {
			t.Fatal("internal state was mutated by external slice modification")
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		p := providers.NewStaticSeedProvider(
			domain.TargetEndpoint{Host: testStaticHost1, Port: 5432},
		)

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := p.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStaticSeedProviderFromStrings(t *testing.T) {
	ctx := context.Background()

	t.Run("valid mixed addresses and comma separated", func(t *testing.T) {
		p, err := providers.NewStaticSeedProviderFromStrings(
			5432,
			"master:5432",
			"replica-1",
			"replica-2:5433, replica-3:5434",
			"[::1]:5435",
			"[::1]",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(seeds) != 6 {
			t.Fatalf("expected 6 seeds, got %d", len(seeds))
		}
	})

	t.Run("invalid port in address", func(t *testing.T) {
		_, err := providers.NewStaticSeedProviderFromStrings(5432, "host:invalid")
		if err == nil {
			t.Fatal("expected error for invalid port, got nil")
		}
	})

	t.Run("empty address list", func(t *testing.T) {
		_, err := providers.NewStaticSeedProviderFromStrings(5432, "   ", "")
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})
}

func TestStaticSeedProvider_AddSeed_And_SetSeeds(t *testing.T) {
	ctx := context.Background()
	p := providers.NewStaticSeedProvider()

	// Проверка AddSeed
	err := p.AddSeed(domain.TargetEndpoint{Host: testStaticHost1, Port: 5432})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Добавление дубликата через AddSeed
	err = p.AddSeed(domain.TargetEndpoint{Host: testStaticHost1, Port: 5432})
	if err != nil {
		t.Fatalf("unexpected error on duplicate: %v", err)
	}

	seeds, _ := p.GetSeeds(ctx)
	if len(seeds) != 1 {
		t.Fatalf("expected 1 seed, got %d", len(seeds))
	}

	// Некорректные параметры AddSeed
	if err := p.AddSeed(domain.TargetEndpoint{Host: "", Port: 5432}); !errors.Is(err, providers.ErrEmptyHostAddress) {
		t.Errorf("expected ErrEmptyHostAddress, got %v", err)
	}
	if err := p.AddSeed(domain.TargetEndpoint{Host: "h", Port: 0}); !errors.Is(err, providers.ErrInvalidPortRange) {
		t.Errorf("expected ErrInvalidPortRange, got %v", err)
	}

	// Проверка SetSeeds
	p.SetSeeds(
		domain.TargetEndpoint{Host: "new1", Port: 5432},
		domain.TargetEndpoint{Host: "new2", Port: 5433},
		domain.TargetEndpoint{Host: "new1", Port: 5432}, // дубликат
	)

	endpoints := p.Endpoints()
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints after SetSeeds, got %d", len(endpoints))
	}

	// Тест на потокобезопасность при конкурентном доступе
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(id int) {
			defer wg.Done()
			_ = p.AddSeed(domain.TargetEndpoint{Host: "concurrent-host", Port: 5000 + id})
		}(i)
		go func() {
			defer wg.Done()
			_, _ = p.GetSeeds(ctx)
		}()
	}
	wg.Wait()
}

func TestSingleEntrypointProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("valid single entrypoint with port", func(t *testing.T) {
		p, err := providers.NewSingleEntrypointProvider("db-master.internal:5432", 5432)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 1 {
			t.Fatalf("expected exactly 1 seed, got %d", len(seeds))
		}

		if seeds[0].Host != "db-master.internal" || seeds[0].Port != 5432 {
			t.Errorf("unexpected seed: %+v", seeds[0])
		}

		ep := p.Endpoint()
		if ep.Host != "db-master.internal" || ep.Port != 5432 {
			t.Errorf("unexpected direct endpoint: %+v", ep)
		}
	})

	t.Run("valid single entrypoint without port uses defaultPort", func(t *testing.T) {
		p, err := providers.NewSingleEntrypointProvider("db-vip.internal", 5432)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 1 || seeds[0].Port != 5432 {
			t.Errorf("expected port 5432, got %+v", seeds)
		}
	})

	t.Run("multiple addresses return ErrSingleEntrypointMultiple", func(t *testing.T) {
		_, err := providers.NewSingleEntrypointProvider("master:5432, replica:5433", 5432)
		if !errors.Is(err, providers.ErrSingleEntrypointMultiple) {
			t.Fatalf("expected ErrSingleEntrypointMultiple, got %v", err)
		}
	})

	t.Run("empty address returns ErrEmptyHostAddress", func(t *testing.T) {
		_, err := providers.NewSingleEntrypointProvider("   ", 5432)
		if !errors.Is(err, providers.ErrEmptyHostAddress) {
			t.Fatalf("expected ErrEmptyHostAddress, got %v", err)
		}
	})

	t.Run("invalid port returns ErrInvalidPortRange", func(t *testing.T) {
		_, err := providers.NewSingleEntrypointProvider("master:99999", 5432)
		if !errors.Is(err, providers.ErrInvalidPortRange) {
			t.Fatalf("expected ErrInvalidPortRange, got %v", err)
		}
	})

	t.Run("context canceled returns context error", func(t *testing.T) {
		p, err := providers.NewSingleEntrypointProvider("master:5432", 5432)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err = p.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestNewEnvSeedProvider(t *testing.T) {
	ctx := context.Background()
	envKey := "TEST_DB_SEEDS_VAR"

	t.Run("successful load from env", func(t *testing.T) {
		t.Setenv(envKey, "10.0.0.1:5432, 10.0.0.2:5433")
		p, err := providers.NewEnvSeedProvider(envKey, 5432)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(seeds) != 2 {
			t.Fatalf("expected 2 seeds, got %d", len(seeds))
		}
	})

	t.Run("empty env returns ErrNoSeedsFound", func(t *testing.T) {
		t.Setenv(envKey, "   ")
		_, err := providers.NewEnvSeedProvider(envKey, 5432)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})
}

func TestViperPrefixSeedProvider_GetSeeds(t *testing.T) {
	ctx := context.Background()

	t.Run("read from viper keys with companion port", func(t *testing.T) {
		v := viper.New()
		v.Set("POSTGRES_HOST_MASTER", "db-master")
		v.Set("POSTGRES_PORT_MASTER", 5432)
		v.Set("POSTGRES_HOST_REPLICA", "db-replica:5433")

		p := providers.NewViperPrefixSeedProvider(v, "POSTGRES_HOST_", 5432)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(seeds) != 2 {
			t.Fatalf("expected 2 seeds, got %d", len(seeds))
		}

		foundMaster := false
		foundReplica := false
		for _, s := range seeds {
			if s.Host == "db-master" && s.Port == 5432 {
				foundMaster = true
			}
			if s.Host == "db-replica" && s.Port == 5433 {
				foundReplica = true
			}
		}
		if !foundMaster || !foundReplica {
			t.Fatalf("missing expected seeds, got: %+v", seeds)
		}
	})

	t.Run("read from os environment matching prefix", func(t *testing.T) {
		t.Setenv("SEED_NODE_PRIMARY", "primary-db")
		t.Setenv("SEED_PORT_PRIMARY", "5432")
		t.Setenv("SEED_NODE_STANDBY", "standby-db:5433")

		p := providers.NewViperPrefixSeedProvider(nil, "SEED_NODE_", 5432)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(seeds) != 2 {
			t.Fatalf("expected 2 seeds, got %d", len(seeds))
		}
	})

	t.Run("no keys found", func(t *testing.T) {
		v := viper.New()
		p := providers.NewViperPrefixSeedProvider(v, "NON_EXISTENT_PREFIX_", 5432)

		_, err := p.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		v := viper.New()
		v.Set("POSTGRES_HOST_PRIMARY", "primary:5432")
		p := providers.NewViperPrefixSeedProvider(v, "POSTGRES_HOST_", 5432)

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := p.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
