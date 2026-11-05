package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/phili/orbit/internal/config"
)

type WSMessage struct {
	Topic     string          `json:"topic"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type ClientAction struct {
	Action string `json:"action"` // "subscribe" | "unsubscribe" | "ping"
	Topic  string `json:"topic"`
}

type clientConn struct {
	id         string
	ws         *websocket.Conn
	hub        *Hub
	sendCh     chan WSMessage
	subs       map[string]bool
	mu         sync.RWMutex
	cancelFunc context.CancelFunc
}

type Hub struct {
	cfg        *config.Config
	log        *slog.Logger
	mu         sync.RWMutex
	clients    map[*clientConn]bool
	topics     map[string]map[*clientConn]bool
	register   chan *clientConn
	unregister chan *clientConn
}

func NewHub(cfg *config.Config, log *slog.Logger) *Hub {
	return &Hub{
		cfg:        cfg,
		log:        log,
		clients:    make(map[*clientConn]bool),
		topics:     make(map[string]map[*clientConn]bool),
		register:   make(chan *clientConn),
		unregister: make(chan *clientConn),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for c := range h.clients {
				c.cancelFunc()
				c.ws.Close(websocket.StatusGoingAway, "server shutting down")
			}
			h.clients = make(map[*clientConn]bool)
			h.topics = make(map[string]map[*clientConn]bool)
			h.mu.Unlock()
			return

		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			h.log.Debug("websocket client connected", "id", c.id)

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				c.mu.RLock()
				for topic := range c.subs {
					if subscribers, ok := h.topics[topic]; ok {
						delete(subscribers, c)
						if len(subscribers) == 0 {
							delete(h.topics, topic)
						}
					}
				}
				c.mu.RUnlock()
				close(c.sendCh)
			}
			h.mu.Unlock()
			h.log.Debug("websocket client disconnected", "id", c.id)
		}
	}
}

func (h *Hub) Broadcast(topic string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		h.log.Error("failed to marshal websocket broadcast", "topic", topic, "err", err)
		return
	}

	msg := WSMessage{
		Topic:     topic,
		Timestamp: time.Now().UnixMilli(),
		Payload:   data,
	}

	h.mu.RLock()
	subscribers, hasTopic := h.topics[topic]
	if !hasTopic || len(subscribers) == 0 {
		h.mu.RUnlock()
		return
	}

	isCritical := strings.HasPrefix(topic, "events") || strings.HasPrefix(topic, "alerts")

	for c := range subscribers {
		select {
		case c.sendCh <- msg:
		default:
			if !isCritical {
				// Non-critical metric frame: drop oldest or skip if channel full
				select {
				case <-c.sendCh: // discard oldest
					c.sendCh <- msg
				default:
					// skip
				}
			} else {
				// Critical message: force enqueue in separate goroutine with timeout
				go func(conn *clientConn, m WSMessage) {
					select {
					case conn.sendCh <- m:
					case <-time.After(2 * time.Second):
						h.log.Warn("dropping critical event due to slow client", "clientId", conn.id)
					}
				}(c, msg)
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Origin verification
	opts := &websocket.AcceptOptions{
		Subprotocols: []string{"orbit.v1"},
	}

	if len(h.cfg.TrustedOrigins) > 0 {
		opts.OriginPatterns = h.cfg.TrustedOrigins
	} else {
		// In dev/local mode allow same host or standard localhost origins
		opts.InsecureSkipVerify = h.cfg.DevMode
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		h.log.Warn("websocket upgrade failed", "err", err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	c := &clientConn{
		id:         r.RemoteAddr,
		ws:         conn,
		hub:        h,
		sendCh:     make(chan WSMessage, 64),
		subs:       make(map[string]bool),
		cancelFunc: cancel,
	}

	h.register <- c

	// Default subscriptions: every client automatically subscribes to "events" and "alerts"
	c.subscribe("events")
	c.subscribe("alerts")

	go c.writePump(ctx)
	c.readPump(ctx)
}

func (c *clientConn) subscribe(topic string) {
	c.mu.Lock()
	c.subs[topic] = true
	c.mu.Unlock()

	c.hub.mu.Lock()
	if _, ok := c.hub.topics[topic]; !ok {
		c.hub.topics[topic] = make(map[*clientConn]bool)
	}
	c.hub.topics[topic][c] = true
	c.hub.mu.Unlock()
}

func (c *clientConn) unsubscribe(topic string) {
	c.mu.Lock()
	delete(c.subs, topic)
	c.mu.Unlock()

	c.hub.mu.Lock()
	if subscribers, ok := c.hub.topics[topic]; ok {
		delete(subscribers, c)
		if len(subscribers) == 0 {
			delete(c.hub.topics, topic)
		}
	}
	c.hub.mu.Unlock()
}

func (c *clientConn) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.cancelFunc()
		c.ws.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var action ClientAction
		err := wsjson.Read(ctx, c.ws, &action)
		if err != nil {
			if errors.Is(err, io.EOF) || websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			if ctx.Err() != nil {
				return
			}
			return
		}

		switch action.Action {
		case "subscribe":
			if action.Topic != "" {
				c.subscribe(action.Topic)
			}
		case "unsubscribe":
			if action.Topic != "" {
				c.unsubscribe(action.Topic)
			}
		case "ping":
			_ = wsjson.Write(ctx, c.ws, map[string]string{"type": "pong"})
		}
	}
}

func (c *clientConn) writePump(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}
			writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
			err := wsjson.Write(writeCtx, c.ws, msg)
			writeCancel()
			if err != nil {
				return
			}

		case <-ticker.C:
			// Ping keepalive
			pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.ws.Ping(pingCtx)
			pingCancel()
			if err != nil {
				return
			}
		}
	}
}

