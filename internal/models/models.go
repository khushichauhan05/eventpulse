package models

import "time"

type Event struct {
	EventID   string  `json:"event_id,omitempty"`
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
}

// MLScoreRequest is sent to the Python ML service.
type MLScoreRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	EventType string  `json:"event_type"`
}

// MLScoreResponse is returned by the Python ML service.
type MLScoreResponse struct {
	RiskScore   int                `json:"risk_score"`
	Confidence  float64            `json:"confidence"`
	IsHighRisk  bool               `json:"is_high_risk"`
	Model       string             `json:"model"`
	Explanation map[string]float64 `json:"explanation"`
}

type ProcessedEvent struct {
	EventID     string             `json:"event_id"`
	UserID      string             `json:"user_id"`
	EventType   string             `json:"event_type"`
	Amount      float64            `json:"amount"`
	RiskScore   int                `json:"risk_score"`
	Confidence  float64            `json:"confidence"`
	MLScored    bool               `json:"ml_scored"`
	Explanation map[string]float64 `json:"explanation,omitempty"`
}

type Alert struct {
	ID          int                `json:"id"`
	EventID     string             `json:"event_id,omitempty"`
	UserID      string             `json:"user_id"`
	RiskScore   int                `json:"risk_score"`
	Confidence  float64            `json:"confidence"`
	Message     string             `json:"message"`
	MLScored    bool               `json:"ml_scored"`
	Explanation map[string]float64 `json:"explanation,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}
