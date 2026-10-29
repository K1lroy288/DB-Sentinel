package http

import (
	"DBSentinel/internal/domain"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// clientSession представляет активное WebSocket-соединение с браузером.
type clientSession struct {
	conn *websocket.Conn
}

// Hub управляет пулом активных WebSocket-клиентов и рассылкой снапшотов топологии.
type Hub struct {
	mu          sync.RWMutex
	clients     map[*clientSession]struct{}
	latestGraph *domain.TopologyGraph
	latestJSON  []byte
}

// NewHub создаёт новый экземпляр Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*clientSession]struct{}),
	}
}

// Register регистрирует новое клиентское соединение, сразу отсылает ему последний снапшот
// и возвращает функцию отмены регистрации при отключении.
func (h *Hub) Register(conn *websocket.Conn) func() {
	session := &clientSession{conn: conn}
	h.mu.Lock()
	h.clients[session] = struct{}{}
	latest := h.latestJSON
	h.mu.Unlock()

	if len(latest) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = conn.Write(ctx, websocket.MessageText, latest)
	}

	return func() {
		h.mu.Lock()
		delete(h.clients, session)
		h.mu.Unlock()
	}
}

// Broadcast рассылает обновленный граф всем подключенным клиентам.
func (h *Hub) Broadcast(graph *domain.TopologyGraph) {
	if graph == nil {
		return
	}

	data, err := json.Marshal(graph)
	if err != nil {
		return
	}

	h.mu.Lock()
	h.latestGraph = graph
	h.latestJSON = data
	sessions := make([]*clientSession, 0, len(h.clients))
	for s := range h.clients {
		sessions = append(sessions, s)
	}
	h.mu.Unlock()

	for _, s := range sessions {
		go func(sess *clientSession) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = sess.conn.Write(ctx, websocket.MessageText, data)
		}(s)
	}
}

// GetLatest возвращает последний сохраненный граф топологии.
func (h *Hub) GetLatest() *domain.TopologyGraph {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.latestGraph
}
