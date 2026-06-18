package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

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

func main() {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "events.raw",
		GroupID: "analytics-group",
	})

	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "events.processed",
		Balancer: &kafka.LeastBytes{},
	}

	fmt.Println("Analytics Service Started...")

	for {

		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Read Error:", err)
			continue
		}

		var event Event

		err = json.Unmarshal(message.Value, &event)
		if err != nil {
			fmt.Println("JSON Error:", err)
			continue
		}

		riskScore := 20

		if event.Amount > 10000 {
			riskScore = 90
		}

		processed := ProcessedEvent{
			UserID:    event.UserID,
			EventType: event.EventType,
			Amount:    event.Amount,
			RiskScore: riskScore,
		}

		processedBytes, _ := json.Marshal(processed)

		err = writer.WriteMessages(
			context.Background(),
			kafka.Message{
				Value: processedBytes,
			},
		)

		if err != nil {
			fmt.Println("Publish Error:", err)
			continue
		}

		fmt.Println("Processed:", string(processedBytes))
	}
}
