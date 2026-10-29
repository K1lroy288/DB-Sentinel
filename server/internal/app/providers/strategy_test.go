package providers_test

import (
	"DBSentinel/internal/app/providers"
	"DBSentinel/internal/domain"
	"context"
	"errors"
	"testing"
)

type mockSeedProvider struct {
	seeds []domain.TargetEndpoint
	err   error
	calls int
}

func (m *mockSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	m.calls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.seeds, m.err
}

func TestFallbackSeedProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("first provider succeeds", func(t *testing.T) {
		p1 := &mockSeedProvider{
			seeds: []domain.TargetEndpoint{{Host: "consul-master", Port: 5432}},
		}
		p2 := &mockSeedProvider{
			seeds: []domain.TargetEndpoint{{Host: "static-master", Port: 5432}},
		}

		fallback := providers.NewFallbackSeedProvider(p1, p2)
		seeds, err := fallback.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 1 || seeds[0].Host != "consul-master" {
			t.Fatalf("expected consul-master, got %+v", seeds)
		}
		if p1.calls != 1 {
			t.Errorf("expected p1 to be called once, got %d", p1.calls)
		}
		if p2.calls != 0 {
			t.Errorf("expected p2 not to be called, got %d", p2.calls)
		}
	})

	t.Run("first provider fails, second succeeds", func(t *testing.T) {
		p1 := &mockSeedProvider{
			err: errors.New("consul unreachable"),
		}
		p2 := &mockSeedProvider{
			seeds: []domain.TargetEndpoint{{Host: "dns-master", Port: 5432}},
		}

		fallback := providers.NewFallbackSeedProvider(p1, p2)
		seeds, err := fallback.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 1 || seeds[0].Host != "dns-master" {
			t.Fatalf("expected dns-master, got %+v", seeds)
		}
		if p1.calls != 1 || p2.calls != 1 {
			t.Errorf("expected both providers called once, got p1=%d, p2=%d", p1.calls, p2.calls)
		}
	})

	t.Run("all providers fail", func(t *testing.T) {
		p1 := &mockSeedProvider{err: errors.New("consul error")}
		p2 := &mockSeedProvider{seeds: []domain.TargetEndpoint{}} // 0 начальных адресов (seeds)

		fallback := providers.NewFallbackSeedProvider(p1, p2)
		_, err := fallback.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrAllProvidersFailed) {
			t.Fatalf("expected ErrAllProvidersFailed, got %v", err)
		}
	})

	t.Run("empty fallback provider returns ErrNoSeedsFound", func(t *testing.T) {
		fallback := providers.NewFallbackSeedProvider()
		_, err := fallback.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})

	t.Run("context canceled returns context error", func(t *testing.T) {
		p1 := &mockSeedProvider{seeds: []domain.TargetEndpoint{{Host: "node1", Port: 5432}}}
		fallback := providers.NewFallbackSeedProvider(p1)

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := fallback.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestCompositeSeedProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("merges and deduplicates endpoints from multiple providers", func(t *testing.T) {
		p1 := &mockSeedProvider{
			seeds: []domain.TargetEndpoint{
				{Host: "master-1", Port: 5432},
				{Host: "replica-1", Port: 5433},
			},
		}
		p2 := &mockSeedProvider{
			seeds: []domain.TargetEndpoint{
				{Host: "replica-1", Port: 5433}, // дубликат
				{Host: "replica-2", Port: 5434},
			},
		}

		composite := providers.NewCompositeSeedProvider(p1, p2)
		seeds, err := composite.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 3 {
			t.Fatalf("expected 3 unique seeds, got %d", len(seeds))
		}

		expected := []string{
			"master-1:5432",
			"replica-1:5433",
			"replica-2:5434",
		}
		for i, exp := range expected {
			if seeds[i].String() != exp {
				t.Errorf("seeds[%d]: expected %s, got %s", i, exp, seeds[i].String())
			}
		}
	})

	t.Run("partial failure returns seeds from healthy providers", func(t *testing.T) {
		p1 := &mockSeedProvider{
			err: errors.New("consul temporary outage"),
		}
		p2 := &mockSeedProvider{
			seeds: []domain.TargetEndpoint{
				{Host: "static-node", Port: 5432},
			},
		}

		composite := providers.NewCompositeSeedProvider(p1, p2)
		seeds, err := composite.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 1 || seeds[0].Host != "static-node" {
			t.Fatalf("expected 1 seed from p2, got %+v", seeds)
		}
	})

	t.Run("all providers fail returns ErrAllProvidersFailed", func(t *testing.T) {
		p1 := &mockSeedProvider{err: errors.New("provider 1 err")}
		p2 := &mockSeedProvider{err: errors.New("provider 2 err")}

		composite := providers.NewCompositeSeedProvider(p1, p2)
		_, err := composite.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrAllProvidersFailed) {
			t.Fatalf("expected ErrAllProvidersFailed, got %v", err)
		}
	})

	t.Run("empty composite provider returns ErrNoSeedsFound", func(t *testing.T) {
		composite := providers.NewCompositeSeedProvider()
		_, err := composite.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})

	t.Run("context canceled returns context error", func(t *testing.T) {
		p1 := &mockSeedProvider{seeds: []domain.TargetEndpoint{{Host: "node1", Port: 5432}}}
		composite := providers.NewCompositeSeedProvider(p1)

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := composite.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
