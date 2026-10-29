package providers

import (
	"DBSentinel/internal/domain"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	_ domain.SeedProvider = (*StaticSeedProvider)(nil)
	_ domain.SeedProvider = (*SingleEntrypointProvider)(nil)
	_ domain.SeedProvider = (*ViperPrefixSeedProvider)(nil)
)

// StaticSeedProvider хранит потокобезопасный предварительно настроенный список целевых адресов базы данных.
type StaticSeedProvider struct {
	mu        sync.RWMutex
	endpoints []domain.TargetEndpoint
}

// NewStaticSeedProvider создаёт новый экземпляр StaticSeedProvider с переданными адресами,
// автоматически выполняя их валидацию и дедупликацию.
func NewStaticSeedProvider(endpoints ...domain.TargetEndpoint) *StaticSeedProvider {
	provider := &StaticSeedProvider{}
	if len(endpoints) > 0 {
		seen := make(map[string]struct{}, len(endpoints))
		deduped := make([]domain.TargetEndpoint, 0, len(endpoints))
		for _, ep := range endpoints {
			if ep.Host == "" || ep.Port <= 0 || ep.Port > 65535 {
				continue
			}
			key := ep.String()
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				deduped = append(deduped, ep)
			}
		}
		provider.endpoints = deduped
	}
	return provider
}

// NewStaticSeedProviderFromStrings разбирает исходные строки адресов (например, "host:port", "host"
// или списки через запятую) и возвращает StaticSeedProvider.
func NewStaticSeedProviderFromStrings(defaultPort int, addresses ...string) (*StaticSeedProvider, error) {
	endpoints, err := NormalizeAndDeduplicate(addresses, defaultPort)
	if err != nil {
		return nil, err
	}
	return NewStaticSeedProvider(endpoints...), nil
}

// NewEnvSeedProvider создаёт StaticSeedProvider, считывая список адресов через запятую
// из указанной переменной окружения (например, "DB_SENTINEL_SEEDS").
func NewEnvSeedProvider(envVar string, defaultPort int) (*StaticSeedProvider, error) {
	val := strings.TrimSpace(os.Getenv(envVar))
	if val == "" {
		return nil, fmt.Errorf("%w: environment variable %q is empty or unset", ErrNoSeedsFound, envVar)
	}
	return NewStaticSeedProviderFromStrings(defaultPort, val)
}

// GetSeeds возвращает защитную копию настроенных целевых адресов.
func (p *StaticSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.endpoints) == 0 {
		return nil, ErrNoSeedsFound
	}

	out := make([]domain.TargetEndpoint, len(p.endpoints))
	copy(out, p.endpoints)
	return out, nil
}

// AddSeed безопасно добавляет новый целевой адрес, если он корректен и ещё отсутствует в списке.
func (p *StaticSeedProvider) AddSeed(endpoint domain.TargetEndpoint) error {
	if endpoint.Host == "" {
		return ErrEmptyHostAddress
	}
	if endpoint.Port <= 0 || endpoint.Port > 65535 {
		return ErrInvalidPortRange
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := endpoint.String()
	for _, ep := range p.endpoints {
		if ep.String() == key {
			return nil // уже существует
		}
	}

	p.endpoints = append(p.endpoints, endpoint)
	return nil
}

// SetSeeds заменяет текущий список адресов новым валидированным списком без дубликатов.
func (p *StaticSeedProvider) SetSeeds(endpoints ...domain.TargetEndpoint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	seen := make(map[string]struct{}, len(endpoints))
	deduped := make([]domain.TargetEndpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		if ep.Host == "" || ep.Port <= 0 || ep.Port > 65535 {
			continue
		}
		key := ep.String()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			deduped = append(deduped, ep)
		}
	}
	p.endpoints = deduped
}

// Endpoints возвращает снимок текущих целевых адресов без необходимости передавать context.
func (p *StaticSeedProvider) Endpoints() []domain.TargetEndpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]domain.TargetEndpoint, len(p.endpoints))
	copy(out, p.endpoints)
	return out
}

// SingleEntrypointProvider хранит ровно одну известную точку входа в базу данных
// (например, адрес балансировщика, VIP или единственный известный мастер-узел), используемую
// в качестве начального зонда для обнаружения остальной топологии.
type SingleEntrypointProvider struct {
	endpoint domain.TargetEndpoint
}

// NewSingleEntrypointProvider анализирует и проверяет, что rawAddress представляет ровно одну целевую точку.
// Возвращает ErrSingleEntrypointMultiple, если передано несколько адресов через запятую.
func NewSingleEntrypointProvider(rawAddress string, defaultPort int) (*SingleEntrypointProvider, error) {
	trimmed := strings.TrimSpace(rawAddress)
	if strings.Contains(trimmed, ",") {
		return nil, ErrSingleEntrypointMultiple
	}

	endpoint, err := ParseEndpoint(trimmed, defaultPort)
	if err != nil {
		return nil, err
	}

	return &SingleEntrypointProvider{
		endpoint: endpoint,
	}, nil
}

// GetSeeds возвращает срез, содержащий единственную целевую точку подключения.
func (p *SingleEntrypointProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return []domain.TargetEndpoint{p.endpoint}, nil
}

// Endpoint возвращает настроенную целевую точку напрямую.
func (p *SingleEntrypointProvider) Endpoint() domain.TargetEndpoint {
	return p.endpoint
}

// ViperPrefixSeedProvider обнаруживает целевые адреса, проверяя конфигурацию Viper
// и переменные окружения, соответствующие заданному префиксу (например, "POSTGRES_HOST_").
type ViperPrefixSeedProvider struct {
	v           *viper.Viper
	prefix      string
	defaultPort int
}

// NewViperPrefixSeedProvider создаёт новый экземпляр ViperPrefixSeedProvider.
func NewViperPrefixSeedProvider(v *viper.Viper, prefix string, defaultPort int) *ViperPrefixSeedProvider {
	if defaultPort <= 0 || defaultPort > 65535 {
		defaultPort = DefaultPostgresPort
	}
	return &ViperPrefixSeedProvider{
		v:           v,
		prefix:      prefix,
		defaultPort: defaultPort,
	}
}

// GetSeeds извлекает, нормализует и дедуплицирует все начальные адреса, соответствующие настроенному префиксу.
func (p *ViperPrefixSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var rawAddresses []string
	prefixLower := strings.ToLower(p.prefix)

	// 1. Сканируем ключи конфигурации Viper (если экземпляр viper передан)
	if p.v != nil {
		for _, key := range p.v.AllKeys() {
			if strings.HasPrefix(strings.ToLower(key), prefixLower) {
				val := strings.TrimSpace(p.v.GetString(key))
				if val != "" {
					rawAddresses = append(rawAddresses, p.enrichAddressWithPort(key, val))
				}
			}
		}
	}

	// 2. Также сканируем переменные окружения ОС (исправляет ситуацию, когда Viper.AutomaticEnv не заполняет AllKeys)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key, val := pair[0], strings.TrimSpace(pair[1])
		if strings.HasPrefix(strings.ToLower(key), prefixLower) && val != "" {
			rawAddresses = append(rawAddresses, p.enrichAddressWithPort(key, val))
		}
	}

	if len(rawAddresses) == 0 {
		return nil, fmt.Errorf("%w: no configuration keys found with prefix %q", ErrNoSeedsFound, p.prefix)
	}

	return NormalizeAndDeduplicate(rawAddresses, p.defaultPort)
}

// enrichAddressWithPort проверяет, есть ли для отдельного значения хоста соответствующий ключ порта
// (например, POSTGRES_PORT_MASTER для POSTGRES_HOST_MASTER).
func (p *ViperPrefixSeedProvider) enrichAddressWithPort(hostKey, val string) string {
	if strings.Contains(val, ",") {
		return val
	}
	if _, _, err := net.SplitHostPort(val); err == nil {
		return val
	}

	portKey := derivePortKey(hostKey)
	if portKey != "" {
		if p.v != nil && p.v.IsSet(portKey) {
			if port := p.v.GetInt(portKey); port > 0 && port <= 65535 {
				return net.JoinHostPort(val, strconv.Itoa(port))
			}
		}
		if envPort := os.Getenv(portKey); envPort != "" {
			if port, err := strconv.Atoi(envPort); err == nil && port > 0 && port <= 65535 {
				return net.JoinHostPort(val, strconv.Itoa(port))
			}
		}
	}

	return val
}

// derivePortKey преобразует имя ключа конфигурации хоста в соответствующий ключ порта
// (например, "POSTGRES_HOST_MASTER" -> "POSTGRES_PORT_MASTER").
func derivePortKey(key string) string {
	if strings.Contains(key, "HOST") {
		return strings.Replace(key, "HOST", "PORT", 1)
	}
	if strings.Contains(key, "host") {
		return strings.Replace(key, "host", "port", 1)
	}
	return ""
}
