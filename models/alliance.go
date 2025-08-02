package models

type AllianceSelection struct {
	AllianceNumber    int    `json:"alliance_number"`
	AllianceCaptain   string `json:"alliance_captain"`
	AllianceSelection string `json:"alliance_selection"`
}
