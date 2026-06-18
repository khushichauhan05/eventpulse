package health

import (
	"encoding/json"
	"net/http"
	"time"
)

type Status struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func NewHandler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Status{
			Service:   serviceName,
			Status:    "ok",
			Timestamp: time.Now().UTC(),
		})
	}
}
