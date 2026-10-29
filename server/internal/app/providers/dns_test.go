package providers_test

import (
	"DBSentinel/internal/app/providers"
	"context"
	"errors"
	"net"
	"testing"
)

type mockSRVResolver struct {
	cname string
	addrs []*net.SRV
	err   error
}

func (m *mockSRVResolver) LookupSRV(ctx context.Context, _, _, _ string) (string, []*net.SRV, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	return m.cname, m.addrs, m.err
}

func TestDNSSRVSeedProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("successful lookup with trailing dot trim", func(t *testing.T) {
		mock := &mockSRVResolver{
			cname: "cluster.internal",
			addrs: []*net.SRV{
				{Target: "db1.node.internal.", Port: 5432},
				{Target: "db2.node.internal.", Port: 5433},
				{Target: "db1.node.internal.", Port: 5432}, // дубликат
			},
		}

		p := providers.NewDNSSRVSeedProvider(
			"postgres", "tcp", "cluster.internal", 5432,
			providers.WithDNSSRVResolver(mock),
		)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 2 {
			t.Fatalf("expected 2 unique seeds, got %d", len(seeds))
		}

		if seeds[0].Host != "db1.node.internal" || seeds[0].Port != 5432 {
			t.Errorf("unexpected seed[0]: %+v", seeds[0])
		}
		if seeds[1].Host != "db2.node.internal" || seeds[1].Port != 5433 {
			t.Errorf("unexpected seed[1]: %+v", seeds[1])
		}
	})

	t.Run("port 0 uses defaultPort", func(t *testing.T) {
		mock := &mockSRVResolver{
			addrs: []*net.SRV{
				{Target: "db-headless.internal.", Port: 0},
			},
		}

		p := providers.NewDNSSRVSeedProvider(
			"", "", "db-headless.internal", 5432,
			providers.WithDNSSRVResolver(mock),
		)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 1 || seeds[0].Port != 5432 {
			t.Errorf("expected port 5432, got %+v", seeds)
		}
	})

	t.Run("resolver error returns error", func(t *testing.T) {
		mock := &mockSRVResolver{
			err: errors.New("dns lookup failed: nxdomain"),
		}

		p := providers.NewDNSSRVSeedProvider(
			"postgres", "tcp", "invalid.domain", 5432,
			providers.WithDNSSRVResolver(mock),
		)

		_, err := p.GetSeeds(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("no SRV records returned returns ErrNoSeedsFound", func(t *testing.T) {
		mock := &mockSRVResolver{
			addrs: []*net.SRV{},
		}

		p := providers.NewDNSSRVSeedProvider(
			"postgres", "tcp", "empty.domain", 5432,
			providers.WithDNSSRVResolver(mock),
		)

		_, err := p.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})

	t.Run("canceled context returns context.Canceled", func(t *testing.T) {
		mock := &mockSRVResolver{
			addrs: []*net.SRV{{Target: "node.internal", Port: 5432}},
		}

		p := providers.NewDNSSRVSeedProvider("postgres", "tcp", "cluster", 5432, providers.WithDNSSRVResolver(mock))

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := p.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
