package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
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

	NumWorkers int `toml:"numWorkers"`
}

type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	IP         string    `json:"ip"`
	StatusCode int       `json:"status_code"`
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Duration   float64   `json:"duration_sec"`
	Service    string    `json:"service"`
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

	switch strings.ToLower(logLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
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

	jobs := make(chan kafka.Message, cfg.NumWorkers*5) // buffer is needed to increase throughput
	var wg sync.WaitGroup
	wg.Add(cfg.NumWorkers)
	for workerID := 0; workerID < cfg.NumWorkers; workerID++ {
		go func(id int) {
			defer wg.Done()
			logWorker(ctx, es, jobs, cfg.ElasticSearchIndex, id)
		}(workerID)
	}

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

		jobs <- msg
	}

	close(jobs)
	wg.Wait()
}

func logWorker(ctx context.Context, es *elasticsearch.Client, jobs <-chan kafka.Message, elasticSearchIndex string, workerID int) {
	for {
		select {
		case <-ctx.Done():
			log.Infof("[logkeeper][workerID:%d] context cancelled, exiting worker", workerID)
			return

		case msg, ok := <-jobs:
			if !ok {
				log.Infof("[logkeeper][workerID:%d] jobs channel closed, exiting worker", workerID)
				return
			}
			log.Debugf("[logkeeper][workerID:%d] received message: %s", workerID, string(msg.Value))

			var entry LogEntry
			if err := json.Unmarshal(msg.Value, &entry); err != nil {
				log.Errorf("[logkeeper][workerID:%d] failed to unmarshal log entry: %v", workerID, err)
				continue
			}

			// Index in Elasticsearch
			res, err := es.Index(
				elasticSearchIndex,
				strings.NewReader(string(msg.Value)),
				es.Index.WithDocumentID(entry.Service+entry.RequestID),
			)
			if res != nil {
				res.Body.Close()
			}
			if err != nil || (res != nil && res.IsError()) {
				log.Errorf("[logkeeper][workerID:%d] failed to index document: %v", workerID, err)
			} else {
				log.Infof("[logkeeper][workerID:%d][%s] log entry indexed", workerID, shorten(entry.RequestID))
			}
		}
	}
}

func shorten(s string) string {
	if len(s) > 6 {
		return s[:6] + "..."
	}
	return s
}
