package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufSize    = 256
)

// WSMessage is the JSON protocol for browser <-> server communication.
type WSMessage struct {
	Type      string         `json:"type"`              // "event", "chat_response", "chat_message", "status", "agent_card"
	Source    string         `json:"source,omitempty"`  // "broker", "cortex", "echo_agent"
	Content   string         `json:"content,omitempty"` // text payload
	Status    string         `json:"status,omitempty"`  // "error" for error events
	Icon      string         `json:"icon,omitempty"`    // "in", "out", "route", "ok", "err", "info"
	Detail    string         `json:"detail,omitempty"`  // secondary info line
	Time      string         `json:"time,omitempty"`    // HH:MM:SS timestamp
	AgentCard *AgentCardInfo `json:"agent_card,omitempty"`
}

// AgentCardInfo carries A2A agent card data to the browser.
type AgentCardInfo struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Version            string            `json:"version"`
	ProtocolVersion    string            `json:"protocol_version,omitempty"`
	URL                string            `json:"url,omitempty"`
	PreferredTransport string            `json:"preferred_transport,omitempty"`
	DocumentationURL   string            `json:"documentation_url,omitempty"`
	IconURL            string            `json:"icon_url,omitempty"`
	Provider           *ProviderInfo     `json:"provider,omitempty"`
	Capabilities       *CapabilitiesInfo `json:"capabilities,omitempty"`
	Skills             []SkillInfo       `json:"skills"`
}

// ProviderInfo describes the agent provider.
type ProviderInfo struct {
	Organization string `json:"organization"`
	URL          string `json:"url,omitempty"`
}

// CapabilitiesInfo describes the agent capabilities.
type CapabilitiesInfo struct {
	Streaming         bool `json:"streaming"`
	PushNotifications bool `json:"push_notifications"`
}

// SkillInfo describes a single agent skill.
type SkillInfo struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	InputModes  []string `json:"input_modes,omitempty"`
	OutputModes []string `json:"output_modes,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const maxHistory = 100 // recent events replayed to new clients

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients    map[*Client]struct{}
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger

	// Event history — replayed to newly connected clients.
	history   [][]byte
	historyMu sync.RWMutex

	// onChatMessage is called when a browser sends a chat_message.
	onChatMessage func(text string)
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
		history:    make([][]byte, 0, maxHistory),
	}
}

// Run starts the hub event loop. It blocks until the broadcast channel is closed.
func (h *Hub) Run() {
	for {
		select {
		case client, ok := <-h.register:
			if !ok {
				return
			}
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()

			// Replay event history to the new client
			h.historyMu.RLock()
			for _, msg := range h.history {
				select {
				case client.send <- msg:
				default:
					// Client buffer full during replay, skip remaining
					break
				}
			}
			h.historyMu.RUnlock()

			h.logger.Info("WebSocket client connected", "clients", len(h.clients))

		case client, ok := <-h.unregister:
			if !ok {
				return
			}
			h.mu.Lock()
			if _, exists := h.clients[client]; exists {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("WebSocket client disconnected", "clients", len(h.clients))

		case message, ok := <-h.broadcast:
			if !ok {
				return
			}
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Slow client — evict
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
					h.mu.RLock()
					h.logger.Warn("evicted slow WebSocket client")
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a WSMessage to all connected clients.
// Events are also stored in history so new clients can catch up.
func (h *Hub) Broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal WSMessage", "error", err)
		return
	}

	// Store event/status messages in replay history
	if msg.Type == "event" || msg.Type == "status" {
		h.historyMu.Lock()
		h.history = append(h.history, data)
		if len(h.history) > maxHistory {
			h.history = h.history[len(h.history)-maxHistory:]
		}
		h.historyMu.Unlock()
	}

	select {
	case h.broadcast <- data:
	default:
		h.logger.Warn("broadcast channel full, dropping message")
	}
}

// ServeWS handles WebSocket upgrade requests.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, sendBufSize)}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

// Client represents a single WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.hub.logger.Error("WebSocket read error", "error", err)
			}
			return
		}
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.hub.logger.Error("invalid WebSocket message", "error", err)
			continue
		}
		if msg.Type == "chat_message" && c.hub.onChatMessage != nil {
			c.hub.onChatMessage(msg.Content)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
