package models

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebSocketResponse struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebSocketPayload struct {
	MatchLevel   string   `json:"match_level"`
	MatchID      int      `json:"match_id"`
	RedAlliance  []string `json:"red_alliance"`
	BlueAlliance []string `json:"blue_alliance"`
}
