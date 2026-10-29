package providers

import (
	"DBSentinel/internal/domain"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var _ domain.SeedProvider = (*DNSSRVSeedProvider)(nil)

// SRVResolver абстрагирует поиск DNS SRV-записей для возможности внедрения моков или пользовательских резолверов.
type SRVResolver interface {
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
}

// DNSSRVOption определяет функциональную опцию для настройки DNSSRVSeedProvider.
type DNSSRVOption func(*DNSSRVSeedProvider)

// WithDNSSRVResolver настраивает пользовательский SRV-резолвер (например, для модульного тестирования).
func WithDNSSRVResolver(resolver SRVResolver) DNSSRVOption {
	return func(p *DNSSRVSeedProvider) {
		if resolver != nil {
			p.resolver = resolver
		}
	}
}

// DNSSRVSeedProvider разрешает целевые адреса баз данных через DNS SRV-записи
// (например, Kubernetes Headless Services или Consul DNS).
type DNSSRVSeedProvider struct {
	service     string
	proto       string
	name        string
	defaultPort int
	resolver    SRVResolver
}

// NewDNSSRVSeedProvider создаёт новый экземпляр DNSSRVSeedProvider.
func NewDNSSRVSeedProvider(service, proto, name string, defaultPort int, opts ...DNSSRVOption) *DNSSRVSeedProvider {
	if defaultPort <= 0 || defaultPort > 65535 {
		defaultPort = DefaultPostgresPort
	}

	p := &DNSSRVSeedProvider{
		service:     service,
		proto:       proto,
		name:        name,
		defaultPort: defaultPort,
		resolver:    net.DefaultResolver,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// GetSeeds опрашивает настроенную DNS SRV-запись и возвращает нормализованные TargetEndpoints.
func (p *DNSSRVSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	_, addrs, err := p.resolver.LookupSRV(ctx, p.service, p.proto, p.name)
	if err != nil {
		targetDesc := p.name
		if p.service != "" && p.proto != "" {
			targetDesc = fmt.Sprintf("_%s._%s.%s", p.service, p.proto, p.name)
		}
		return nil, fmt.Errorf("failed to lookup SRV records for %s: %w", targetDesc, err)
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("%w: no SRV records returned", ErrNoSeedsFound)
	}

	raw := make([]string, 0, len(addrs))
	for _, srv := range addrs {
		target := strings.TrimSpace(strings.TrimSuffix(srv.Target, "."))
		if target == "" {
			continue
		}

		port := int(srv.Port)
		if port <= 0 {
			port = p.defaultPort
		}

		raw = append(raw, net.JoinHostPort(target, strconv.Itoa(port)))
	}

	return NormalizeAndDeduplicate(raw, p.defaultPort)
}
