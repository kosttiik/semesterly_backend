package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type ProgressUpdate struct {
	Type           string  `json:"type"`
	CurrentItem    int     `json:"currentItem"`
	TotalItems     int     `json:"totalItems"`
	CompletedItems int     `json:"completedItems"`
	Message        string  `json:"message"`
	Percentage     float64 `json:"percentage"`
	ETA            string  `json:"eta"`
}

type Client struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
}

type WebSocketHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256), // Добавляем буферизированный канал для избежания блокировки
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *WebSocketHub) BroadcastProgress(update ProgressUpdate) {
	h.mu.Lock()
	if len(h.clients) == 0 {
		h.mu.Unlock()
		return // Нет подключенных клиентов, сворачиваем работу
	}
	h.mu.Unlock()

	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling progress update: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
		// Сообщение успешно отправлено
	default:
		log.Printf("Broadcast channel is full, progress update dropped")
	}
}

func (h *WebSocketHub) CloseAll() {
	h.mu.Lock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.clients = make(map[*Client]bool)
	h.mu.Unlock()

	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			c.conn.Close()
			close(c.send)
		}(client)
	}
	wg.Wait()
}

func (a *App) HandleWebSocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := &Client{
		hub:  a.Hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	client.hub.register <- client

	go client.writePump()

	return nil
}

func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for message := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error writing to websocket: %v", err)
			return
		}
	}
}
