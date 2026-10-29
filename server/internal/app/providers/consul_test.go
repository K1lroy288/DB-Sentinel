package providers_test

import (
	"DBSentinel/internal/app/providers"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConsulSeedProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("successful lookup with passing instances and node fallback", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/health/service/postgres-cluster" {
				t.Errorf("unexpected path: %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			if r.URL.Query().Get("passing") != "true" {
				t.Errorf("expected passing=true query param")
			}
			if r.URL.Query().Get("dc") != "dc-primary" {
				t.Errorf("expected dc=dc-primary query param, got %s", r.URL.Query().Get("dc"))
			}
			if r.Header.Get("X-Consul-Token") != "secret-token" {
				t.Errorf("expected X-Consul-Token header")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{
					"Node": {"Address": "192.168.1.10"},
					"Service": {"Address": "10.0.0.1", "Port": 5432}
				},
				{
					"Node": {"Address": "192.168.1.11"},
					"Service": {"Address": "", "Port": 5433}
				}
			]`))
		}))
		defer ts.Close()

		p := providers.NewConsulSeedProvider(
			ts.URL,
			"postgres-cluster",
			5432,
			providers.WithConsulToken("secret-token"),
			providers.WithConsulDatacenter("dc-primary"),
			providers.WithConsulHTTPClient(ts.Client()),
		)

		seeds, err := p.GetSeeds(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seeds) != 2 {
			t.Fatalf("expected 2 seeds, got %d", len(seeds))
		}

		if seeds[0].Host != "10.0.0.1" || seeds[0].Port != 5432 {
			t.Errorf("unexpected seed[0]: %+v", seeds[0])
		}
		// Адрес узла (Node) используется, если Service.Address пустой
		if seeds[1].Host != "192.168.1.11" || seeds[1].Port != 5433 {
			t.Errorf("unexpected seed[1]: %+v", seeds[1])
		}
	})

	t.Run("empty passing nodes returns ErrNoSeedsFound", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}))
		defer ts.Close()

		p := providers.NewConsulSeedProvider(
			ts.URL,
			"empty-cluster",
			5432,
			providers.WithConsulHTTPClient(ts.Client()),
		)

		_, err := p.GetSeeds(ctx)
		if !errors.Is(err, providers.ErrNoSeedsFound) {
			t.Fatalf("expected ErrNoSeedsFound, got %v", err)
		}
	})

	t.Run("consul non-200 response returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		p := providers.NewConsulSeedProvider(
			ts.URL,
			"failing-cluster",
			5432,
			providers.WithConsulHTTPClient(ts.Client()),
		)

		_, err := p.GetSeeds(ctx)
		if err == nil {
			t.Fatal("expected error on 500 response, got nil")
		}
	})

	t.Run("consul malformed JSON returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not-json}`))
		}))
		defer ts.Close()

		p := providers.NewConsulSeedProvider(
			ts.URL,
			"corrupt-cluster",
			5432,
			providers.WithConsulHTTPClient(ts.Client()),
		)

		_, err := p.GetSeeds(ctx)
		if err == nil {
			t.Fatal("expected decode error, got nil")
		}
	})

	t.Run("context canceled returns context error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}))
		defer ts.Close()

		p := providers.NewConsulSeedProvider(
			ts.URL,
			"postgres-cluster",
			5432,
			providers.WithConsulHTTPClient(ts.Client()),
		)

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := p.GetSeeds(canceledCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
