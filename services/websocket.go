package services

import (
	"log"
	"net/http"
	"sync"

	"github.com/Jake-Schuler/ORC-MatchMaker/models"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		log.Printf("WebSocket upgrade check origin: %s", r.Header.Get("Origin"))
		return true // Allow all origins for simplicity; adjust as needed
	},
}

func HandleWebSocketConnection(conn *websocket.Conn) {
	defer func() {
		Manager.RemoveConnection(conn)
		conn.Close()
	}()

	log.Println("WebSocket connection established")

	// Add connection to manager
	Manager.AddConnection(conn)

	// Handle WebSocket messages in a loop
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("Received message: %s", message)

		// Echo the message back (for now)
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}

	log.Println("WebSocket connection closed")
}

func SetMatchLevel(conn *websocket.Conn, level string) error {
	return conn.WriteJSON(models.WebSocketMessage{
		Type:    "set_match_level",
		Payload: level,
	})
}

func SendMatchUpdate(conn *websocket.Conn, payload models.WebSocketPayload) error {
	return conn.WriteJSON(models.WebSocketMessage{
		Type:    "match_update",
		Payload: payload,
	})
}

// WebSocket connection manager
type ConnectionManager struct {
	connections map[*websocket.Conn]bool
	mutex       sync.RWMutex
}

var Manager = &ConnectionManager{
	connections: make(map[*websocket.Conn]bool),
}

// Add connection to manager
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.connections[conn] = true
	log.Printf("WebSocket connection added. Total connections: %d", len(cm.connections))
}

// Remove connection from manager
func (cm *ConnectionManager) RemoveConnection(conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.connections, conn)
	log.Printf("WebSocket connection removed. Total connections: %d", len(cm.connections))
}

// Broadcast message to all connected clients
func (cm *ConnectionManager) Broadcast(message models.WebSocketMessage) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for conn := range cm.connections {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			// Remove dead connection
			go func(c *websocket.Conn) {
				cm.RemoveConnection(c)
				c.Close()
			}(conn)
		}
	}
}

// Broadcast active match update to all connected clients
func BroadcastActiveMatch(matchLevel string, matchID int, redPlayerID string, bluePlayerID string, db *gorm.DB) {
	var redPlayer, bluePlayer models.User

	// Find users by MMID
	db.Where("mm_id = ?", redPlayerID).First(&redPlayer)
	db.Where("mm_id = ?", bluePlayerID).First(&bluePlayer)

	// Use preferred username if available, otherwise fall back to username
	redUsername := redPlayer.PreferedUsername
	if redUsername == "" {
		redUsername = redPlayer.Username
	}

	blueUsername := bluePlayer.PreferedUsername
	if blueUsername == "" {
		blueUsername = bluePlayer.Username
	}

	payload := models.WebSocketPayload{
		MatchLevel:   matchLevel,
		MatchID:      matchID,
		RedAlliance:  []string{redUsername},
		BlueAlliance: []string{blueUsername},
	}
	message := models.WebSocketMessage{
		Type:    "active_match_update",
		Payload: payload,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted active match update: Level=%s, ID=%d, Red=%s, Blue=%s", matchLevel, matchID, redUsername, blueUsername)
}
