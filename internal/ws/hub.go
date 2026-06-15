package ws

import (
    "encoding/json"
    "net/http"
    "sync"

    "catan-backend/internal/model"
    "github.com/gorilla/websocket"
)

type Client struct {
    conn   *websocket.Conn
    send   chan []byte
    gameID string
}

type Hub struct {
    mu      sync.RWMutex
    clients map[string]map[*Client]bool // gameID -> clients
    upgrader websocket.Upgrader
}

func NewHub() *Hub {
    return &Hub{
        clients: make(map[string]map[*Client]bool),
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }
}

func (h *Hub) register(c *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    m, ok := h.clients[c.gameID]
    if !ok {
        m = make(map[*Client]bool)
        h.clients[c.gameID] = m
    }
    m[c] = true
}

func (h *Hub) unregister(c *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if m, ok := h.clients[c.gameID]; ok {
        if _, ok := m[c]; ok {
            delete(m, c)
            close(c.send)
        }
        if len(m) == 0 {
            delete(h.clients, c.gameID)
        }
    }
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, gameID string) {
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    client := &Client{conn: conn, send: make(chan []byte, 16), gameID: gameID}
    h.register(client)

    // writer
    go func() {
        for data := range client.send {
            _ = conn.WriteMessage(websocket.TextMessage, data)
        }
        conn.Close()
    }()

    // reader (drains and closes on error)
    go func() {
        defer h.unregister(client)
        for {
            if _, _, err := conn.NextReader(); err != nil {
                return
            }
        }
    }()
}

func (h *Hub) BroadcastGame(game *model.Game) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    data, err := json.Marshal(map[string]*model.Game{"game": game})
    if err != nil {
        return
    }
    if clients, ok := h.clients[game.ID]; ok {
        for c := range clients {
            select {
            case c.send <- data:
            default:
                // if client's send channel is full, drop and unregister
                go h.unregister(c)
            }
        }
    }
}

// package-level default hub for convenience
var defaultHub *Hub

func SetDefaultHub(h *Hub) {
    defaultHub = h
}

func BroadcastGame(game *model.Game) {
    if defaultHub == nil {
        return
    }
    defaultHub.BroadcastGame(game)
}
