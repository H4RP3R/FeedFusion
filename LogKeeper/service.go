package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	LogLevel     string   `toml:"logLevel"`
	KafkaBrokers []string `toml:"kafkaBrokers"`
	KafkaTopic   string   `toml:"kafkaTopic"`
	KafkaGroupID string   `toml:"kafkaGroupID"`

	ElasticSearchIndex string   `toml:"elasticSearchIndex"`
	ElasticSearchNodes []string `toml:"elasticSearchNodes"`
}

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
	var (
		configPath string
		logLevel   string
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("[logkeeper] shutting down gracefully...")
		cancel()
	}()

	flag.StringVar(&configPath, "config", "config.toml", "Path to TOML config file")
	flag.StringVar(&logLevel, "log", "info", "Log level: debug, info, warn, error.")
	flag.Parse()

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		log.Fatalf("[server] failed to load config file %s: %v", configPath, err)
	}

	// Override config with flags if set
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: cfg.ElasticSearchNodes})
	if err != nil {
		log.Fatalf("[logkeeper] error creating the client: %s", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    cfg.KafkaTopic,
		GroupID:  cfg.KafkaGroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer r.Close()

	log.Info("[logkeeper] accepting logs...")
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			log.Errorf("[logkeeper] failed to read message from Kafka: %v", err)
			continue
		}
		log.Debugf("[logkeeper] received message: %s", string(msg.Value))

		var entry LogEntry
		if err := json.Unmarshal(msg.Value, &entry); err != nil {
			log.Errorf("[logkeeper] failed to unmarshal log entry: %v", err)
			continue
		}

		// Index in Elasticsearch
		res, err := es.Index(
			cfg.ElasticSearchIndex,
			strings.NewReader(string(msg.Value)),
			es.Index.WithDocumentID(entry.RequestID),
		)
		if res != nil {
			defer res.Body.Close()
		}
		if err != nil || (res != nil && res.IsError()) {
			log.Errorf("[logkeeper] failed to index document: %v", err)
		} else {
			log.Infof("[logkeeper][%s] log entry indexed", shorten(entry.RequestID))
		}
	}
}

func shorten(s string) string {
	if len(s) > 6 {
		return s[:6] + "..."
	}
	return s
}
