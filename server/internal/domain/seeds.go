// Package domain определяет основные доменные сущности и интерфейсы для DB-Sentinel.
package domain

import (
	"context"
	"net"
	"strconv"
)

// TargetEndpoint представляет нормализованный сетевой адрес базы данных (хост и TCP-порт).
type TargetEndpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// String возвращает строковое представление адреса в формате host:port.
func (e TargetEndpoint) String() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// SeedProvider — основной интерфейс для обнаружения начальных точек подключения к базе данных.
type SeedProvider interface {
	// GetSeeds обнаруживает, нормализует и возвращает список целевых адресов баз данных.
	GetSeeds(ctx context.Context) ([]TargetEndpoint, error)
}
