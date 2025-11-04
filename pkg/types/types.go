package types

type JoinRequest struct {
	PlayerID string  `json:"player_id"`
	MMR      float64 `json:"mmr"`
}

type Player struct {
	PlayerID string
	MMR      float64
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Match struct {
	Players []Player `json:"players"`
}