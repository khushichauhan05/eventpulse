package models

import "time"

type Event struct {
	EventID   string  `json:"event_id,omitempty"`
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
}

type ProcessedEvent struct {
	EventID   string  `json:"event_id"`
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
	RiskScore int     `json:"risk_score"`
}

type Alert struct {
	ID        int       `json:"id"`
	EventID   string    `json:"event_id,omitempty"`
	UserID    string    `json:"user_id"`
	RiskScore int       `json:"risk_score"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
