package providers

import (
	"DBSentinel/internal/domain"
	"context"
	"fmt"
	"strings"
)

var (
	_ domain.SeedProvider = (*FallbackSeedProvider)(nil)
	_ domain.SeedProvider = (*CompositeSeedProvider)(nil)
)

// FallbackSeedProvider последовательно опрашивает список SeedProvider в порядке резервирования.
// Возвращается результат первого провайдера, вернувшего хотя бы один корректный адрес без ошибок.
// Если провайдер вернул ошибку или 0 адресов, опрашивается следующий провайдер в цепочке.
type FallbackSeedProvider struct {
	providers []domain.SeedProvider
}

// NewFallbackSeedProvider создаёт новый экземпляр FallbackSeedProvider с переданными провайдерами.
// Провайдеры со значением nil игнорируются.
func NewFallbackSeedProvider(providers ...domain.SeedProvider) *FallbackSeedProvider {
	filtered := make([]domain.SeedProvider, 0, len(providers))
	for _, p := range providers {
		if p != nil {
			filtered = append(filtered, p)
		}
	}
	return &FallbackSeedProvider{
		providers: filtered,
	}
}

// GetSeeds поочерёдно опрашивает каждого провайдера, пока один из них не вернёт непустой результат.
func (f *FallbackSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(f.providers) == 0 {
		return nil, ErrNoSeedsFound
	}

	var errMessages []string
	for i, provider := range f.providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		seeds, err := provider.GetSeeds(ctx)
		if err == nil && len(seeds) > 0 {
			return seeds, nil
		}

		if err != nil {
			errMessages = append(errMessages, fmt.Sprintf("provider[%d]: %v", i, err))
		} else {
			errMessages = append(errMessages, fmt.Sprintf("provider[%d]: returned 0 seeds", i))
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrAllProvidersFailed, strings.Join(errMessages, "; "))
}

// CompositeSeedProvider опрашивает несколько SeedProvider, агрегирует и дедуплицирует
// все обнаруженные целевые адреса в единый общий список.
type CompositeSeedProvider struct {
	providers []domain.SeedProvider
}

// NewCompositeSeedProvider создаёт новый экземпляр CompositeSeedProvider. Провайдеры со значением nil игнорируются.
func NewCompositeSeedProvider(providers ...domain.SeedProvider) *CompositeSeedProvider {
	filtered := make([]domain.SeedProvider, 0, len(providers))
	for _, p := range providers {
		if p != nil {
			filtered = append(filtered, p)
		}
	}
	return &CompositeSeedProvider{
		providers: filtered,
	}
}

// GetSeeds опрашивает все зарегистрированные провайдеры, собирая и дедуплицируя их адреса.
// Если все провайдеры вернули ошибку, возвращается ErrAllProvidersFailed с подробностями.
func (c *CompositeSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(c.providers) == 0 {
		return nil, ErrNoSeedsFound
	}

	seen := make(map[string]struct{})
	var combined []domain.TargetEndpoint
	var errMessages []string

	for i, provider := range c.providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		seeds, err := provider.GetSeeds(ctx)
		if err != nil {
			errMessages = append(errMessages, fmt.Sprintf("provider[%d]: %v", i, err))
			continue
		}

		for _, s := range seeds {
			key := s.String()
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				combined = append(combined, s)
			}
		}
	}

	if len(combined) == 0 {
		if len(errMessages) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrAllProvidersFailed, strings.Join(errMessages, "; "))
		}
		return nil, ErrNoSeedsFound
	}

	return combined, nil
}
