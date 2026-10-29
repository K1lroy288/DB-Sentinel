package http_test

import (
	"DBSentinel/internal/domain"
	internalHttp "DBSentinel/internal/transport/http"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"
)

const testServerAddr = ":8080"

func TestServer_Health(t *testing.T) {
	server := internalHttp.NewServer(internalHttp.Config{Addr: testServerAddr}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", res["status"])
	}
}

func TestServer_TopologyEndpoint(t *testing.T) {
	hub := internalHttp.NewHub()
	server := internalHttp.NewServer(internalHttp.Config{Addr: testServerAddr}, nil, hub)
	handler := server.Handler()

	t.Run("returns 503 when no topology collected yet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topology", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("returns 200 with graph after broadcast", func(t *testing.T) {
		graph := &domain.TopologyGraph{
			ClusterName: "test-cluster",
			Timestamp:   domain.NewUnixMillisTime(time.Now()),
			Nodes: []domain.DatabaseNode{
				{
					ID:       "pg-master:5432",
					Hostname: "pg-master",
					Role:     domain.RolePrimary,
					Status:   domain.StatusHealthy,
				},
			},
		}
		hub.Broadcast(graph)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/topology", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var received domain.TopologyGraph
		if err := json.Unmarshal(w.Body.Bytes(), &received); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if received.ClusterName != "test-cluster" {
			t.Errorf("expected cluster name 'test-cluster', got %s", received.ClusterName)
		}
	})
}

func TestServer_StaticAndSPAFallback(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><html><body>Root SPA</body></html>")},
		"assets/app.js": {Data: []byte("console.log('app');")},
	}

	server := internalHttp.NewServer(internalHttp.Config{Addr: testServerAddr}, mockFS, nil)
	handler := server.Handler()

	t.Run("serves index.html on root /", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if body := w.Body.String(); body != "<!doctype html><html><body>Root SPA</body></html>" {
			t.Errorf("unexpected body content: %s", body)
		}
	})

	t.Run("serves static asset when file exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if body := w.Body.String(); body != "console.log('app');" {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("falls back to index.html for SPA routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nodes/master-1", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if body := w.Body.String(); body != "<!doctype html><html><body>Root SPA</body></html>" {
			t.Errorf("unexpected body: %s", body)
		}
	})
}
