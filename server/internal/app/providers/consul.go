package providers

import (
	"DBSentinel/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var _ domain.SeedProvider = (*ConsulSeedProvider)(nil)

// DefaultConsulTimeout — тайм-аут HTTP-запроса по умолчанию для вызовов Consul API.
const DefaultConsulTimeout = 3 * time.Second

// ConsulOption определяет функциональную опцию для настройки ConsulSeedProvider.
type ConsulOption func(*ConsulSeedProvider)

// WithConsulHTTPClient настраивает пользовательский *http.Client.
func WithConsulHTTPClient(client *http.Client) ConsulOption {
	return func(p *ConsulSeedProvider) {
		if client != nil {
			p.httpClient = client
		}
	}
}

// WithConsulToken задаёт токен ACL, передаваемый в заголовке X-Consul-Token.
func WithConsulToken(token string) ConsulOption {
	return func(p *ConsulSeedProvider) {
		p.token = strings.TrimSpace(token)
	}
}

// WithConsulDatacenter настраивает параметр запроса дата-центра (dc=...).
func WithConsulDatacenter(dc string) ConsulOption {
	return func(p *ConsulSeedProvider) {
		p.datacenter = strings.TrimSpace(dc)
	}
}

// WithConsulTimeout переопределяет тайм-аут HTTP-запроса по умолчанию.
func WithConsulTimeout(timeout time.Duration) ConsulOption {
	return func(p *ConsulSeedProvider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

// ConsulSeedProvider запрашивает HTTP API HashiCorp Consul для получения работоспособных экземпляров сервиса.
type ConsulSeedProvider struct {
	consulAddr  string
	serviceName string
	defaultPort int
	httpClient  *http.Client
	token       string
	datacenter  string
}

type consulHealthEntry struct {
	Node struct {
		Address string `json:"Address"`
	} `json:"Node"`
	Service struct {
		Address string `json:"Address"`
		Port    int    `json:"Port"`
	} `json:"Service"`
}

// NewConsulSeedProvider создаёт новый экземпляр ConsulSeedProvider, настроенный с помощью функциональных опций.
func NewConsulSeedProvider(consulAddr, serviceName string, defaultPort int, opts ...ConsulOption) *ConsulSeedProvider {
	addr := strings.TrimSpace(consulAddr)
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	addr = strings.TrimSuffix(addr, "/")

	if defaultPort <= 0 || defaultPort > 65535 {
		defaultPort = DefaultPostgresPort
	}

	p := &ConsulSeedProvider{
		consulAddr:  addr,
		serviceName: strings.TrimSpace(serviceName),
		defaultPort: defaultPort,
		httpClient: &http.Client{
			Timeout: DefaultConsulTimeout,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// GetSeeds запрашивает API Consul /v1/health/service/:service?passing=true и возвращает работоспособные адреса.
func (p *ConsulSeedProvider) GetSeeds(ctx context.Context) ([]domain.TargetEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	endpointURL := fmt.Sprintf("%s/v1/health/service/%s?passing=true", p.consulAddr, url.PathEscape(p.serviceName))
	if p.datacenter != "" {
		endpointURL += fmt.Sprintf("&dc=%s", url.QueryEscape(p.datacenter))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build consul request: %w", err)
	}

	if p.token != "" {
		req.Header.Set("X-Consul-Token", p.token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consul API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("consul API returned non-200 code: %d", resp.StatusCode)
	}

	var entries []consulHealthEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode consul response: %w", err)
	}

	var raw []string
	for _, entry := range entries {
		host := strings.TrimSpace(entry.Service.Address)
		if host == "" {
			host = strings.TrimSpace(entry.Node.Address)
		}
		if host == "" {
			continue
		}

		port := entry.Service.Port
		if port <= 0 {
			port = p.defaultPort
		}

		raw = append(raw, net.JoinHostPort(host, strconv.Itoa(port)))
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: consul returned 0 passing instances for service %q", ErrNoSeedsFound, p.serviceName)
	}

	return NormalizeAndDeduplicate(raw, p.defaultPort)
}
