package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/Jake-Schuler/ORC-MatchMaker/models"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var event_name = "Online Robotics Competition"
var leaderboard_visible = false        // Track leaderboard visibility state
var alliance_selection_visible = false // Track alliance selection visibility state

// SetEventName updates the global event name
func SetEventName(name string) {
	event_name = name
}

// GetEventName returns the current event name
func GetEventName() string {
	return event_name
}

// GetLeaderboardVisibility returns the current leaderboard visibility state
func GetLeaderboardVisibility() bool {
	return leaderboard_visible
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		log.Printf("WebSocket upgrade check origin: %s", r.Header.Get("Origin"))
		return true // Allow all origins for simplicity; adjust as needed
	},
}

func HandleWebSocketConnection(conn *websocket.Conn, db *gorm.DB) {
	defer func() {
		Manager.RemoveConnection(conn)
		conn.Close()
	}()

	log.Println("WebSocket connection established")

	// Add connection to manager
	Manager.AddConnection(conn)

	// Handle WebSocket messages in a loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("Received message: %s", message)

		// Parse JSON message
		var wsMessage models.WebSocketMessage
		err = json.Unmarshal(message, &wsMessage)
		if err != nil {
			log.Printf("Error parsing WebSocket message: %v", err)
			continue
		}

		if wsMessage.Type == "statusbar_init" {
			// Send initial status bar data
			statusBarData := models.WebSocketMatchPayload{
				RedAlliance:  []string{""},
				BlueAlliance: []string{""},
				EventName:    event_name,
				MatchLevel:   "",
				MatchID:      0,
			}
			response := models.WebSocketMessage{
				Type:    "active_match_update",
				Payload: statusBarData,
			}
			err = conn.WriteJSON(response)
			if err != nil {
				log.Printf("WebSocket write error for status bar: %v", err)
				break
			}
			log.Println("Sent initial status bar data")
		} else if wsMessage.Type == "request_available_teams" {
			// Send available teams data
			availableTeams := GetAvailableTeams(db)
			response := models.WebSocketMessage{
				Type: "available_teams_update",
				Payload: map[string]interface{}{
					"teams": availableTeams,
				},
			}
			err = conn.WriteJSON(response)
			if err != nil {
				log.Printf("WebSocket write error for available teams: %v", err)
				break
			}
			log.Println("Sent available teams data")
		} else if wsMessage.Type == "team_selected" {
			// Handle team selection from client
			var payload map[string]interface{}
			payloadBytes, _ := json.Marshal(wsMessage.Payload)
			json.Unmarshal(payloadBytes, &payload)

			if username, ok := payload["username"].(string); ok {
				// Broadcast team selection to all clients
				BroadcastTeamSelection(username)
				log.Printf("Broadcasted team selection: %s", username)
			}
		}
	}

	log.Println("WebSocket connection closed")
} // WebSocket connection manager
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

	payload := models.WebSocketMatchPayload{
		MatchLevel:   matchLevel,
		MatchID:      matchID,
		EventName:    event_name,
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

func BroadcastLeaderboardUpdate(db *gorm.DB) {
	leaderboard, err := GetLeaderboard(db)
	if err != nil {
		log.Printf("Error getting leaderboard for broadcast: %v", err)
		return
	}

	message := models.WebSocketMessage{
		Type:    "leaderboard_update",
		Payload: leaderboard,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted leaderboard update: %d users", len(leaderboard))
}

func ToggleLeaderboardVisibility() {
	leaderboard_visible = !leaderboard_visible // Toggle the global state
	payload := models.WebSocketLeaderboardTogglePayload{
		Show: leaderboard_visible,
	}
	message := models.WebSocketMessage{
		Type:    "leaderboard_toggle",
		Payload: payload,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted leaderboard visibility toggle: %v", leaderboard_visible)
}

func EndScreenBroadcast(redAlliance []string, blueAlliance []string) {
	payload := models.WebSocketMatchSavedPayload{
		RedAlliance:  redAlliance,
		BlueAlliance: blueAlliance,
	}
	message := models.WebSocketMessage{
		Type:    "match_saved",
		Payload: payload,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted end screen match saved: Red=%v, Blue=%v", redAlliance, blueAlliance)
}

func BroadcastAllianceSelection(allianceSelection models.AllianceSelection) {
	payload := models.WebSocketAllianceSelectionPayload{
		AllianceNumber:    allianceSelection.AllianceNumber,
		AllianceCaptain:   allianceSelection.AllianceCaptain,
		AllianceSelection: allianceSelection.AllianceSelection,
	}
	message := models.WebSocketMessage{
		Type:    "alliance_selection",
		Payload: payload,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted alliance selection: %d, Captain=%s, Selection=%s",
		allianceSelection.AllianceNumber, allianceSelection.AllianceCaptain, allianceSelection.AllianceSelection)
}

func ToggleAllianceSelectionVisibility() {
	alliance_selection_visible = !alliance_selection_visible
	payload := models.WebSocketToggleAllianceSlectionPayload{
		Show: alliance_selection_visible,
	}
	message := models.WebSocketMessage{
		Type:    "alliance_selection_toggle",
		Payload: payload,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted alliance selection visibility toggle: %v", alliance_selection_visible)
}

// GetAvailableTeams returns users that haven't been selected for alliances yet
func GetAvailableTeams(db *gorm.DB) []models.User {
	var allUsers []models.User
	var selectedUsers []string

	// Get all users from database
	if err := db.Find(&allUsers).Error; err != nil {
		log.Printf("Error fetching users: %v", err)
		return []models.User{}
	}

	// Calculate RP and stats for each user to determine rank
	for i := range allUsers {
		winRP, bonusRP, AutoPoints, TeleopPoints, EndgamePoints := calculateUserStats(db, allUsers[i].MMID)
		allUsers[i].WinRP = winRP
		allUsers[i].BonusRP = bonusRP
		allUsers[i].TotalRP = winRP + bonusRP
		allUsers[i].TotalPoints = AutoPoints + TeleopPoints + EndgamePoints
		allUsers[i].AutoPoints = AutoPoints
		allUsers[i].TeleopPoints = TeleopPoints
		allUsers[i].EndgamePoints = EndgamePoints
	}

	// Sort users by TotalRP (descending) to assign ranks
	for i := 0; i < len(allUsers)-1; i++ {
		for j := i + 1; j < len(allUsers); j++ {
			if allUsers[i].TotalRP < allUsers[j].TotalRP {
				allUsers[i], allUsers[j] = allUsers[j], allUsers[i]
			}
		}
	}

	// Assign ranks based on the sorted order
	for i := range allUsers {
		allUsers[i].Rank = i + 1
	}

	// Get all alliance selections to find already selected users
	var allianceSelections []models.AllianceSelection
	if err := db.Find(&allianceSelections).Error; err != nil {
		log.Printf("Error fetching alliance selections: %v", err)
		// If we can't get alliance selections, return all users
		return allUsers
	}

	// Collect all selected usernames (both captains and picks)
	for _, selection := range allianceSelections {
		if selection.AllianceCaptain != "" {
			selectedUsers = append(selectedUsers, selection.AllianceCaptain)
		}
		if selection.AllianceSelection != "" {
			selectedUsers = append(selectedUsers, selection.AllianceSelection)
		}
	}

	// Filter out selected users
	var availableUsers []models.User
	for _, user := range allUsers {
		username := user.PreferedUsername
		if username == "" {
			username = user.Username
		}

		// Check if this user is already selected
		isSelected := false
		for _, selectedUser := range selectedUsers {
			if selectedUser == username {
				isSelected = true
				break
			}
		}

		// If not selected, add to available list
		if !isSelected {
			availableUsers = append(availableUsers, user)
		}
	}

	log.Printf("Found %d available teams out of %d total users", len(availableUsers), len(allUsers))
	return availableUsers
}

// BroadcastTeamSelection notifies all clients that a team has been selected
func BroadcastTeamSelection(username string) {
	payload := map[string]interface{}{
		"username": username,
	}
	message := models.WebSocketMessage{
		Type:    "team_selection_made",
		Payload: payload,
	}
	Manager.Broadcast(message)
	log.Printf("Broadcasted team selection: %s", username)
}
