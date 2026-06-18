package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type Event struct {
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
}

type Alert struct {
	ID        int    `json:"id"`
	UserID    string `json:"user_id"`
	RiskScore int    `json:"risk_score"`
	Message   string `json:"message"`
}

var writer = &kafka.Writer{
	Addr:     kafka.TCP("localhost:9092"),
	Topic:    "events.raw",
	Balancer: &kafka.LeastBytes{},
}

var db *sql.DB

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "EventPulse API Running")
}

func createEvent(w http.ResponseWriter, r *http.Request) {
	var event Event

	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	eventBytes, _ := json.Marshal(event)

	err = writer.WriteMessages(
		context.Background(),
		kafka.Message{
			Value: eventBytes,
		},
	)

	if err != nil {
		fmt.Println("Kafka Error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("Published Event:", string(eventBytes))

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Event Published"))
}

func getAlerts(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(
		`SELECT id, user_id, risk_score, message
		 FROM alerts
		 ORDER BY id DESC`,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var alerts []Alert

	for rows.Next() {

		var alert Alert

		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.RiskScore,
			&alert.Message,
		)

		if err != nil {
			continue
		}

		alerts = append(alerts, alert)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func main() {
	db, _ = sql.Open(
		"postgres",
		"host=localhost port=5432 user=admin password=admin123 dbname=eventpulse sslmode=disable",
	)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/events", createEvent)
	http.HandleFunc("/alerts", getAlerts)
	fmt.Println("Server started on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
