package providers

import (
	"DBSentinel/internal/domain"
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ProviderType определяет тип источника сидов.
type ProviderType string

// Поддерживаемые типы сид-провайдеров.
const (
	ProviderTypeStatic   ProviderType = "static"
	ProviderTypeSingle   ProviderType = "single"
	ProviderTypeConsul   ProviderType = "consul"
	ProviderTypeDNS      ProviderType = "dns"
	ProviderTypeEnv      ProviderType = "env"
	ProviderTypeFallback ProviderType = "fallback"
)

// ResolverConfig содержит параметры для создания и настройки сид-провайдера.
type ResolverConfig struct {
	ProviderType    ProviderType
	StaticAddresses []string
	SingleAddress   string
	EnvVarName      string
	ConsulAddr      string
	ConsulService   string
	ConsulToken     string
	ConsulDC        string
	DNSService      string
	DNSProto        string
	DNSName         string
	DefaultPort     int

	// Опциональные зависимости для тестирования
	ConsulClient *http.Client
	DNSResolver  SRVResolver
}

// SeedResolver — фасадная сущность, управляющая инициализацией и получением сидов на основе конфигурации.
type SeedResolver struct {
	provider domain.SeedProvider
}

// NewSeedResolver создаёт новый экземпляр SeedResolver на основе переданной конфигурации.
func NewSeedResolver(cfg ResolverConfig) (*SeedResolver, error) {
	if cfg.DefaultPort <= 0 || cfg.DefaultPort > 65535 {
		cfg.DefaultPort = DefaultPostgresPort
	}

	pType := ProviderType(strings.ToLower(strings.TrimSpace(string(cfg.ProviderType))))
	if pType == "" {
		if cfg.SingleAddress != "" {
			pType = ProviderTypeSingle
		} else {
			pType = ProviderTypeStatic
		}
	}

	provider, err := createProvider(pType, cfg)
	if err != nil {
		return nil, err
	}

	return &SeedResolver{
		provider: provider,
	}, nil
}

func createProvider(pType ProviderType, cfg ResolverConfig) (domain.SeedProvider, error) {
	switch pType {
	case ProviderTypeStatic:
		return buildStaticProvider(cfg)
	case ProviderTypeSingle:
		return NewSingleEntrypointProvider(cfg.SingleAddress, cfg.DefaultPort)
	case ProviderTypeEnv:
		return NewEnvSeedProvider(cfg.EnvVarName, cfg.DefaultPort)
	case ProviderTypeConsul:
		return buildConsulProvider(cfg)
	case ProviderTypeDNS:
		return buildDNSProvider(cfg)
	case ProviderTypeFallback:
		return buildFallbackProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported seed provider type %q", cfg.ProviderType)
	}
}

func buildStaticProvider(cfg ResolverConfig) (domain.SeedProvider, error) {
	if len(cfg.StaticAddresses) == 0 {
		return nil, fmt.Errorf("%w: static addresses must not be empty", ErrNoSeedsFound)
	}
	return NewStaticSeedProviderFromStrings(cfg.DefaultPort, cfg.StaticAddresses...)
}

func buildConsulProvider(cfg ResolverConfig) (domain.SeedProvider, error) {
	if cfg.ConsulAddr == "" || cfg.ConsulService == "" {
		return nil, fmt.Errorf("consulAddr and consulService are required for consul provider")
	}
	var opts []ConsulOption
	if cfg.ConsulClient != nil {
		opts = append(opts, WithConsulHTTPClient(cfg.ConsulClient))
	}
	if cfg.ConsulToken != "" {
		opts = append(opts, WithConsulToken(cfg.ConsulToken))
	}
	if cfg.ConsulDC != "" {
		opts = append(opts, WithConsulDatacenter(cfg.ConsulDC))
	}
	return NewConsulSeedProvider(cfg.ConsulAddr, cfg.ConsulService, cfg.DefaultPort, opts...), nil
}

func buildDNSProvider(cfg ResolverConfig) (domain.SeedProvider, error) {
	if cfg.DNSName == "" {
		return nil, fmt.Errorf("dnsName is required for dns provider")
	}
	var opts []DNSSRVOption
	if cfg.DNSResolver != nil {
		opts = append(opts, WithDNSSRVResolver(cfg.DNSResolver))
	}
	return NewDNSSRVSeedProvider(cfg.DNSService, cfg.DNSProto, cfg.DNSName, cfg.DefaultPort, opts...), nil
}

func buildFallbackProvider(cfg ResolverConfig) (domain.SeedProvider, error) {
	var chain []domain.SeedProvider

	if cfg.ConsulAddr != "" && cfg.ConsulService != "" {
		if cp, err := buildConsulProvider(cfg); err == nil {
			chain = append(chain, cp)
		}
	}

	if cfg.DNSName != "" {
		if dp, err := buildDNSProvider(cfg); err == nil {
			chain = append(chain, dp)
		}
	}

	if len(cfg.StaticAddresses) > 0 {
		if sp, err := buildStaticProvider(cfg); err == nil {
			chain = append(chain, sp)
		}
	} else if cfg.SingleAddress != "" {
		if singleP, err := NewSingleEntrypointProvider(cfg.SingleAddress, cfg.DefaultPort); err == nil {
			chain = append(chain, singleP)
		}
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("fallback strategy requires at least one configured provider")
	}

	return NewFallbackSeedProvider(chain...), nil
}

// Provider возвращает сконфигурированный экземпляр domain.SeedProvider.
func (r *SeedResolver) Provider() domain.SeedProvider {
	return r.provider
}

// ResolveSeeds запускает процесс получения и нормализации сидов из сконфигурированного источника.
func (r *SeedResolver) ResolveSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if r.provider == nil {
		return nil, fmt.Errorf("seed provider is not configured")
	}

	seeds, err := r.provider.GetSeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve seeds: %w", err)
	}

	if len(seeds) == 0 {
		return nil, ErrNoSeedsFound
	}

	return seeds, nil
}
