package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/segmentio/kafka-go"
)

type Event struct {
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
}

var writer = &kafka.Writer{
	Addr:     kafka.TCP("localhost:9092"),
	Topic:    "events.raw",
	Balancer: &kafka.LeastBytes{},
}

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

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/events", createEvent)

	fmt.Println("Server started on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
