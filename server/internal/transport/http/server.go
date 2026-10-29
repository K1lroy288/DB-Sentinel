package http

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Server реализует HTTP и WebSocket веб-сервер для отдачи Web UI и потоковых данных топологии.
type Server struct {
	httpServer *http.Server
	hub        *Hub
	staticFS   fs.FS
}

// Config конфигурация HTTP-сервера.
type Config struct {
	Addr string
}

// NewServer создаёт и инициализирует веб-сервер.
func NewServer(cfg Config, staticFS fs.FS, hub *Hub) *Server {
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if hub == nil {
		hub = NewHub()
	}

	s := &Server{
		hub:      hub,
		staticFS: staticFS,
	}

	mux := http.NewServeMux()

	// REST API
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/topology", s.handleTopology)

	// WebSocket Gateway
	mux.HandleFunc("/ws", s.handleWebSocket)

	// SPA & Static Assets
	mux.HandleFunc("/", s.handleStatic)

	s.httpServer = &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

// Start запускает прослушивание входящих HTTP/WS соединений.
func (s *Server) Start() error {
	log.Printf("HTTP & WebSocket server listening on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown корректно останавливает сервер.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Handler возвращает внутренний http.Handler сервера.
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

// Hub возвращает хаб рассылки топологии.
func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleTopology(w http.ResponseWriter, _ *http.Request) {
	graph := s.hub.GetLatest()
	w.Header().Set("Content-Type", "application/json")

	if graph == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"topology not yet collected"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(graph)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket handshake error: %v", err)
		return
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "bye")
	}()

	unregister := s.hub.Register(conn)
	defer unregister()

	// Читаем входящие сообщения (например, ping/pong) пока клиент не разорвет соединение
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.staticFS == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<h1>DB-Sentinel Core</h1><p>Web UI static assets not embedded.</p>"))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Проверяем, существует ли запрашиваемый файл
	file, err := s.staticFS.Open(path)
	if err != nil {
		// Для SPA отдаем index.html на любой неопознанный клиентский маршрут
		indexBytes, indexErr := fs.ReadFile(s.staticFS, "index.html")
		if indexErr != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBytes)
		return
	}
	_ = file.Close()

	http.FileServer(http.FS(s.staticFS)).ServeHTTP(w, r)
}
