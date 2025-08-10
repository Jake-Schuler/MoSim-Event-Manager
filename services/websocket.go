package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/Jake-Schuler/MoSim-Event-Manager/models"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var event_name = "Online Robotics Competition"
var leaderboard_visible = false        // Track leaderboard visibility state
var alliance_selection_visible = false // Track alliance selection visibility state

// Global state variables to persist WebSocket state
var current_match_state *models.WebSocketMatchPayload
var current_leaderboard_state []models.User
var current_alliance_selections []models.AllianceSelection

// SetEventName updates the global event name
func SetEventName(name string) {
	event_name = name
	// Update current match state if it exists
	if current_match_state != nil {
		current_match_state.EventName = name
	}
}

// GetEventName returns the current event name
func GetEventName() string {
	return event_name
}

// GetLeaderboardVisibility returns the current leaderboard visibility state
func GetLeaderboardVisibility() bool {
	return leaderboard_visible
}

func ResetAllianceSelections() {
	current_alliance_selections = nil
	log.Println("Reset current alliance selections")
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
			// Send initial status bar data - use stored state if available
			var statusBarData models.WebSocketMatchPayload
			if current_match_state != nil {
				statusBarData = *current_match_state
			} else {
				statusBarData = models.WebSocketMatchPayload{
					RedAlliance:  []string{""},
					BlueAlliance: []string{""},
					EventName:    event_name,
					MatchLevel:   "",
					MatchID:      0,
				}
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

			// Send current leaderboard state if available
			if current_leaderboard_state != nil {
				leaderboardResponse := models.WebSocketMessage{
					Type:    "leaderboard_update",
					Payload: current_leaderboard_state,
				}
				err = conn.WriteJSON(leaderboardResponse)
				if err != nil {
					log.Printf("WebSocket write error for leaderboard: %v", err)
				} else {
					log.Println("Sent stored leaderboard data")
				}
			}

			// Send current leaderboard visibility state
			leaderboardToggle := models.WebSocketMessage{
				Type: "leaderboard_toggle",
				Payload: models.WebSocketLeaderboardTogglePayload{
					Show: leaderboard_visible,
				},
			}
			err = conn.WriteJSON(leaderboardToggle)
			if err != nil {
				log.Printf("WebSocket write error for leaderboard toggle: %v", err)
			} else {
				log.Printf("Sent leaderboard visibility state: %v", leaderboard_visible)
			}

			// Send current alliance selection visibility state
			allianceToggle := models.WebSocketMessage{
				Type: "alliance_selection_toggle",
				Payload: models.WebSocketToggleAllianceSlectionPayload{
					Show: alliance_selection_visible,
				},
			}
			err = conn.WriteJSON(allianceToggle)
			if err != nil {
				log.Printf("WebSocket write error for alliance toggle: %v", err)
			} else {
				log.Printf("Sent alliance selection visibility state: %v", alliance_selection_visible)
			}

			// Send current alliance selections if any exist
			if len(current_alliance_selections) > 0 {
				for _, selection := range current_alliance_selections {
					if selection.AllianceCaptain != "" || selection.AllianceSelection != "" {
						allianceResponse := models.WebSocketMessage{
							Type: "alliance_selection",
							Payload: models.WebSocketAllianceSelectionPayload{
								AllianceNumber:    selection.AllianceNumber,
								AllianceCaptain:   selection.AllianceCaptain,
								AllianceSelection: selection.AllianceSelection,
							},
						}
						err = conn.WriteJSON(allianceResponse)
						if err != nil {
							log.Printf("WebSocket write error for alliance selection: %v", err)
						}
					}
				}
				log.Println("Sent stored alliance selection data")
			}
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

	// Store the current match state
	current_match_state = &payload

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

	// Store the current leaderboard state
	current_leaderboard_state = leaderboard

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

	// Clear the current match state since the match has ended
	ClearMatchState()
}

func BroadcastAllianceSelection(allianceSelection models.AllianceSelection) {
	payload := models.WebSocketAllianceSelectionPayload{
		AllianceNumber:    allianceSelection.AllianceNumber,
		AllianceCaptain:   allianceSelection.AllianceCaptain,
		AllianceSelection: allianceSelection.AllianceSelection,
	}

	// Update current alliance selections state
	updateCurrentAllianceSelections(allianceSelection)

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

// updateCurrentAllianceSelections updates the current alliance selections state
func updateCurrentAllianceSelections(newSelection models.AllianceSelection) {
	if current_alliance_selections == nil {
		current_alliance_selections = make([]models.AllianceSelection, 0)
	}

	// Find existing alliance selection or add new one
	found := false
	for i, selection := range current_alliance_selections {
		if selection.AllianceNumber == newSelection.AllianceNumber {
			current_alliance_selections[i] = newSelection
			found = true
			break
		}
	}

	if !found {
		current_alliance_selections = append(current_alliance_selections, newSelection)
	}
}

// InitializeWebSocketState loads the current state from the database
func InitializeWebSocketState(db *gorm.DB) {
	log.Println("Initializing WebSocket state from database...")

	// Load current alliance selections
	var allianceSelections []models.AllianceSelection
	if err := db.Find(&allianceSelections).Error; err != nil {
		log.Printf("Error loading alliance selections: %v", err)
	} else {
		current_alliance_selections = allianceSelections
		log.Printf("Loaded %d alliance selections", len(allianceSelections))
	}

	// Load current leaderboard
	if leaderboard, err := GetLeaderboard(db); err == nil {
		current_leaderboard_state = leaderboard
		log.Printf("Loaded leaderboard with %d users", len(leaderboard))
	} else {
		log.Printf("Error loading leaderboard: %v", err)
	}

	log.Println("WebSocket state initialization complete")
}

// ClearMatchState clears the current match state (useful when match ends)
func ClearMatchState() {
	current_match_state = nil
	current_alliance_selections = nil
	log.Println("Cleared current match state and alliance selections")
}
