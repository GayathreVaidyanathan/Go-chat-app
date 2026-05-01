package handlers

import (
	"context"
	"encoding/json"
	"go-chat/middleware"
	"go-chat/models"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn     *websocket.Conn
	room     string
	username string
	send     chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	broadcast  chan *RoomMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	db         *mongo.Database
	jwtSecret  string
}

type RoomMessage struct {
	room    string
	message []byte
}

func NewHub(db *mongo.Database, jwtSecret string) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan *RoomMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		db:         db,
		jwtSecret:  jwtSecret,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if h.rooms[client.room] == nil {
				h.rooms[client.room] = make(map[*Client]bool)
			}
			h.rooms[client.room][client] = true
			h.mu.Unlock()
			h.broadcastRoomUsers(client.room)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.rooms[client.room], client)
				close(client.send)
			}
			h.mu.Unlock()
			h.broadcastRoomUsers(client.room)

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.rooms[msg.room] {
				select {
				case client.send <- msg.message:
				default:
					close(client.send)
					delete(h.clients, client)
					delete(h.rooms[msg.room], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) broadcastRoomUsers(room string) {
	h.mu.RLock()
	users := []string{}
	for client := range h.rooms[room] {
		users = append(users, client.username)
	}
	h.mu.RUnlock()

	msg, _ := json.Marshal(map[string]interface{}{
		"type":  "users",
		"users": users,
	})

	h.mu.RLock()
	for client := range h.rooms[room] {
		select {
		case client.send <- msg:
		default:
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	room := r.URL.Query().Get("room")
	if room == "" {
		room = "General"
	}

	_, username, err := middleware.GetUserFromToken(token, h.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		conn:     conn,
		room:     room,
		username: username,
		send:     make(chan []byte, 256),
	}

	h.register <- client

	// Send message history
	history, _ := h.getHistory(room)
	for _, msg := range history {
		data, _ := json.Marshal(map[string]interface{}{
			"type":       "message",
			"username":   msg.Username,
			"text":       msg.Text,
			"room":       msg.Room,
			"created_at": msg.CreatedAt,
		})
		client.send <- data
	}

	// Send join notification
	joinMsg, _ := json.Marshal(map[string]interface{}{
		"type":     "message",
		"username": "System",
		"text":     username + " joined the room 🎉",
		"room":     room,
	})
	h.broadcast <- &RoomMessage{room: room, message: joinMsg}

	go client.writePump()
	go client.readPump(h)
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) readPump(h *Hub) {
	defer func() {
		leaveMsg, _ := json.Marshal(map[string]interface{}{
			"type":     "message",
			"username": "System",
			"text":     c.username + " left the room 👋",
			"room":     c.room,
		})
		h.broadcast <- &RoomMessage{room: c.room, message: leaveMsg}
		h.unregister <- c
		c.conn.Close()
	}()

	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
			continue
		}

		// Save to MongoDB
		message := models.Message{
			Room:      c.room,
			Username:  c.username,
			Text:      wsMsg.Text,
			CreatedAt: time.Now(),
		}
		h.db.Collection("messages").InsertOne(context.Background(), message)

		// Broadcast
		data, _ := json.Marshal(map[string]interface{}{
			"type":       "message",
			"username":   c.username,
			"text":       wsMsg.Text,
			"room":       c.room,
			"created_at": message.CreatedAt,
		})
		h.broadcast <- &RoomMessage{room: c.room, message: data}
	}
}

func (h *Hub) getHistory(room string) ([]models.Message, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(50)
	cursor, err := h.db.Collection("messages").Find(context.Background(), bson.M{"room": room}, opts)
	if err != nil {
		return nil, err
	}
	var messages []models.Message
	cursor.All(context.Background(), &messages)

	// Reverse
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}