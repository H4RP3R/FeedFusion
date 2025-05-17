package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	IP         string    `json:"ip"`
	StatusCode int       `json:"status_code"`
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Duration   float64   `json:"duration_sec"`
}

func main() {
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{"http://localhost:9200"}})
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "feed-fusion-logs",
		GroupID:  "logkeeper-group",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	for {
		msg, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka read error: %v", err)
			continue
		}
		log.Infof("Received message: %s", string(msg.Value))

		var entry LogEntry
		if err := json.Unmarshal(msg.Value, &entry); err != nil {
			log.Printf("Failed to parse log entry: %v", err)
			continue
		}

		// Index in Elasticsearch
		res, err := es.Index(
			"feed-fusion-logs",
			strings.NewReader(string(msg.Value)),
			es.Index.WithDocumentID(entry.RequestID),
		)

		if err != nil || res.IsError() {
			log.Printf("Failed to index document: %v", err)
		}
	}
}
