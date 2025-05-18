package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"gateway/pkg/api"
)

type Config struct {
	ServiceName string                 `toml:"serviceName"`
	SubServices map[string]api.Service `toml:"subServices"`

	HTTPAddr   string `toml:"httpAddr"`
	LogLevel   string `toml:"logLevel"`
	KafkaAddr  string `toml:"kafkaAddr"`
	KafkaTopic string `toml:"kafkaTopic"`
	KafkaBatch int    `toml:"kafkaBatch"`
}

func main() {
	var (
		configPath string
		httpAddr   string
		logLevel   string
		kafkaAddr  string
		kafkaTopic string
		kafkaBatch int
	)

	flag.StringVar(&configPath, "config", "cmd/server/config.toml", "Path to TOML config file")
	flag.StringVar(&httpAddr, "http", ":8088", "HTTP server address in the form 'host:port'.")
	flag.StringVar(&logLevel, "log", "info", "Log level: debug, info, warn, error.")
	flag.StringVar(&kafkaAddr, "kafka", "", "Kafka server address in the form 'host:port'.")
	flag.StringVar(&kafkaTopic, "topic", "", "Kafka topic.")
	flag.IntVar(&kafkaBatch, "batch", 0, "Kafka batch size.")
	flag.Parse()

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		log.Fatalf("[server] failed to load config file %s: %v", configPath, err)
	}

	// Override config with flags if set
	if httpAddr != "" {
		cfg.HTTPAddr = httpAddr
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if kafkaAddr != "" {
		cfg.KafkaAddr = kafkaAddr
	}
	if kafkaTopic != "" {
		cfg.KafkaTopic = kafkaTopic
	}
	if kafkaBatch != 0 {
		cfg.KafkaBatch = kafkaBatch
	}

	if !strings.Contains(httpAddr, ":") {
		log.Warn("[server] use ':' before port number, e.g. ':8080'")
	}

	switch logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

	var kafkaWriter *kafka.Writer
	if cfg.KafkaAddr != "" && cfg.KafkaTopic != "" {
		kafkaWriter = &kafka.Writer{
			Addr:      kafka.TCP(cfg.KafkaAddr),
			Topic:     cfg.KafkaTopic,
			BatchSize: cfg.KafkaBatch,
		}
		err := createTopic(kafkaWriter.Addr.String(), kafkaWriter.Topic)
		if err != nil {
			log.Warnf("[server] failed to create Kafka topic: %v", err)
		}
	} else {
		log.Warnf("[server] kafka was not configured, logs will not be sent to Kafka")
	}

	api, err := api.New(cfg.ServiceName, cfg.SubServices, kafkaWriter)
	if err != nil {
		log.Fatalf("[server] failed to create API: %v", err)
	}

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: api.Router(),
	}

	go func() {
		log.Infof("[server] starting on port %v", httpAddr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[server] failed to start: %v", err)
			return
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorf("[server] HTTP server shutdown error: %v", err)
	} else {
		log.Info("[server] HTTP server shut down gracefully")
	}
}

func createTopic(broker, topic string) error {
	conn, err := kafka.DialContext(context.Background(), "tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
}
