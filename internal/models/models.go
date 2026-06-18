package models

import "time"

type Event struct {
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
}

type ProcessedEvent struct {
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
	RiskScore int     `json:"risk_score"`
}

type Alert struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	RiskScore int       `json:"risk_score"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
