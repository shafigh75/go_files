package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Message is the structure sent between server and client.
type Message struct {
	Type     string   `json:"type"`     // "message", "join", "leave", or "users"
	Username string   `json:"username"` // Sender or affected user.
	Room     string   `json:"room"`     // Chat room.
	Message  string   `json:"message,omitempty"`
	Users    []string `json:"users,omitempty"`
}

// Client is a user connected via WebSocket.
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	username string
	room     string
}

// Hub holds all active rooms and clients.
type Hub struct {
	rooms      map[string]map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

// NewHub creates and returns a new Hub.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan Message, 10000),
		register:   make(chan *Client, 1000),
		unregister: make(chan *Client, 1000),
	}
}

func (h *Hub) run() {
	for {
		select {
		// Register new client.
		case client := <-h.register:
			h.mu.Lock()
			clients, ok := h.rooms[client.room]
			if !ok {
				clients = make(map[*Client]bool)
				h.rooms[client.room] = clients
			}
			clients[client] = true
			h.mu.Unlock()

			// Send join message and update user list.
			joinMsg := Message{
				Type:     "join",
				Username: client.username,
				Room:     client.room,
				Message:  fmt.Sprintf("%s joined the room", client.username),
			}
			h.broadcast <- joinMsg
			h.sendUserList(client.room)

		// Unregister client.
		case client := <-h.unregister:
			clients, ok := h.rooms[client.room]
			if ok {
				if _, present := clients[client]; present {
					h.mu.Lock()
					delete(clients, client)
					close(client.send)
					leaveMsg := Message{
						Type:     "leave",
						Username: client.username,
						Room:     client.room,
						Message:  fmt.Sprintf("%s left the room", client.username),
					}
					h.broadcast <- leaveMsg
					h.mu.Unlock()
					h.sendUserList(client.room)
					if len(clients) == 0 {
						h.mu.Lock()
						delete(h.rooms, client.room)
						h.mu.Unlock()
					}
				}
			}

		// Broadcast messages to all clients in the same room.
		case msg := <-h.broadcast:
			h.mu.Lock()
			if clients, ok := h.rooms[msg.Room]; ok {
				data, err := json.Marshal(msg)
				if err != nil {
					log.Printf("Error marshaling: %v", err)
					h.mu.Unlock()
					break
				}
				for client := range clients {
					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

// sendUserList collects all active usernames in a room and broadcasts them.
func (h *Hub) sendUserList(room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var users []string
	if clients, ok := h.rooms[room]; ok {
		for client := range clients {
			users = append(users, client.username)
		}
	}
	userMsg := Message{
		Type:  "users",
		Room:  room,
		Users: users,
	}
	if clients, ok := h.rooms[room]; ok {
		data, err := json.Marshal(userMsg)
		if err != nil {
			log.Printf("Error marshaling users: %v", err)
			return
		}
		for client := range clients {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// For demo purposes; in production check the origin.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// serveWs upgrades HTTP requests to a WebSocket connection.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Expect username and room as URL query parameters.
	username := r.URL.Query().Get("username")
	room := r.URL.Query().Get("room")
	if username == "" || room == "" {
		http.Error(w, "username and room required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WS: %v", err)
		return
	}
	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		username: username,
		room:     room,
	}
	hub.register <- client

	// Start write and read goroutines.
	go client.writePump()
	go client.readPump(hub)
}

// readPump reads messages from the WebSocket and broadcasts them.
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}
		outMsg := Message{
			Type:     "message",
			Username: c.username,
			Room:     c.room,
			Message:  string(msg),
		}
		hub.broadcast <- outMsg
	}
}

// writePump writes messages from the hub into the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed by hub.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write any queued messages.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// Send a ping.
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func main() {
	hub := NewHub()
	go hub.run()

	router := mux.NewRouter()
	// Serve the chat page.
	router.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})
	// WebSocket endpoint.
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	// Serve the login page.
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/login.html")
	})
	// Serve any other static files.
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	addr := ":8080"
	fmt.Printf("Server started on %s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
