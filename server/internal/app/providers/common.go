// Package providers реализует провайдеры начальных адресов (seed providers)
// для обнаружения и настройки целевых узлов базы данных при мониторинге кластера.
package providers

import (
	"DBSentinel/internal/domain"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// DefaultPostgresPort — порт TCP по умолчанию для экземпляров PostgreSQL.
const DefaultPostgresPort = 5432

// Общие ошибки, возвращаемые провайдерами начальных адресов.
var (
	// ErrNoSeedsFound указывает, что не найдено и не настроено ни одного корректного начального адреса.
	ErrNoSeedsFound = errors.New("no seed endpoints found")

	// ErrEmptyHostAddress указывает, что переданная строка адреса хоста пуста.
	ErrEmptyHostAddress = errors.New("empty host address")

	// ErrInvalidPortRange указывает, что номер порта находится вне допустимого диапазона TCP (1-65535).
	ErrInvalidPortRange = errors.New("port out of range (must be 1-65535)")

	// ErrSingleEntrypointMultiple указывает, что передано несколько адресов там, где разрешён ровно один.
	ErrSingleEntrypointMultiple = errors.New("single entrypoint provider requires exactly one address")

	// ErrAllProvidersFailed указывает, что все опрошенные провайдеры в fallback- или composite-стратегии завершились с ошибкой.
	ErrAllProvidersFailed = errors.New("all seed providers failed")
)

// ParseEndpoint разбирает строку адреса (например, "host", "host:port", "[::1]:5432", "[::1]")
// и возвращает валидированный domain.TargetEndpoint, используя defaultPort, если порт не указан.
func ParseEndpoint(raw string, defaultPort int) (domain.TargetEndpoint, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"'`)
	if raw == "" {
		return domain.TargetEndpoint{}, ErrEmptyHostAddress
	}

	if defaultPort <= 0 || defaultPort > 65535 {
		defaultPort = DefaultPostgresPort
	}

	var host string
	var port int

	if h, portStr, err := net.SplitHostPort(raw); err == nil {
		host = h
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			return domain.TargetEndpoint{}, fmt.Errorf("%w: %q", ErrInvalidPortRange, portStr)
		}
		port = p
	} else {
		// Обработка IPv6-хостов в квадратных скобках без порта: "[::1]"
		if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
			host = strings.Trim(raw, "[]")
		} else {
			host = raw
		}
		port = defaultPort
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return domain.TargetEndpoint{}, ErrEmptyHostAddress
	}

	return domain.TargetEndpoint{
		Host: host,
		Port: port,
	}, nil
}

// NormalizeAndDeduplicate разбирает, нормализует и удаляет дубликаты из среза строк адресов.
// Также разделяет элементы, перечисленные через запятую внутри одной строки.
func NormalizeAndDeduplicate(rawAddresses []string, defaultPort int) ([]domain.TargetEndpoint, error) {
	if defaultPort <= 0 || defaultPort > 65535 {
		defaultPort = DefaultPostgresPort
	}

	seen := make(map[string]struct{})
	var result []domain.TargetEndpoint

	for _, raw := range rawAddresses {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		parts := strings.Split(raw, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			endpoint, err := ParseEndpoint(part, defaultPort)
			if err != nil {
				return nil, fmt.Errorf("invalid endpoint %q: %w", part, err)
			}

			key := endpoint.String()
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				result = append(result, endpoint)
			}
		}
	}

	if len(result) == 0 {
		return nil, ErrNoSeedsFound
	}

	return result, nil
}
