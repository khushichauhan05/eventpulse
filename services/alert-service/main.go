package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type ProcessedEvent struct {
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
	RiskScore int     `json:"risk_score"`
}

type Alert struct {
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	RiskScore int    `json:"risk_score"`
}

func main() {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "events.processed",
		GroupID: "alert-group",
	})

	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "alerts",
		Balancer: &kafka.LeastBytes{},
	}

	fmt.Println("Alert Service Started...")

	for {

		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Read Error:", err)
			continue
		}

		var event ProcessedEvent

		err = json.Unmarshal(message.Value, &event)
		if err != nil {
			fmt.Println("JSON Error:", err)
			continue
		}

		if event.RiskScore >= 80 {

			alert := Alert{
				UserID:    event.UserID,
				Message:   "HIGH RISK TRANSACTION DETECTED",
				RiskScore: event.RiskScore,
			}

			alertBytes, _ := json.Marshal(alert)

			err = writer.WriteMessages(
				context.Background(),
				kafka.Message{
					Value: alertBytes,
				},
			)

			if err != nil {
				fmt.Println("Alert Publish Error:", err)
				continue
			}

			fmt.Println("ALERT GENERATED:")
			fmt.Println(string(alertBytes))
		}
	}
}
