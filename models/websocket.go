package models

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebSocketResponse struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebSocketMatchPayload struct {
	MatchLevel   string   `json:"match_level"`
	MatchID      int      `json:"match_id"`
	EventName    string   `json:"event_name"`
	RedAlliance  []string `json:"red_alliance"`
	BlueAlliance []string `json:"blue_alliance"`
}

type WebSocketLeaderboardPayload struct {
	Users []User `json:"users"`
}

type WebSocketLeaderboardTogglePayload struct {
	Show bool `json:"show"`
}

type WebSocketMatchSavedPayload struct {
	RedAlliance  []string `json:"red_alliance"`
	BlueAlliance []string `json:"blue_alliance"`
}

type WebSocketAllianceSelectionPayload struct {
	AllianceNumber    int    `json:"alliance_number"`
	AllianceCaptain   string `json:"alliance_captain"`
	AllianceSelection string `json:"alliance_selection"`
}

type WebSocketToggleAllianceSlectionPayload struct {
	Show bool `json:"show"`
}
