package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"news/pkg/api"
	"news/pkg/rss"
	"news/pkg/storage"
	"news/pkg/storage/memdb"
	"news/pkg/storage/postgres"
)

type Config struct {
	ServiceName string `toml:"serviceName"`
	HTTPAddr    string `toml:"httpAddr"`
	LogLevel    string `toml:"logLevel"`
	KafkaAddr   string `toml:"kafkaAddr"`
	KafkaTopic  string `toml:"kafkaTopic"`
	KafkaBatch  int    `toml:"kafkaBatch"`
}

func main() {
	var (
		sdb storage.Storage
		dev bool

		configPath string
		httpAddr   string
		logLevel   string
		kafkaAddr  string
		kafkaTopic string
		kafkaBatch int
	)

	var (
		sigChan = make(chan os.Signal, 1)
		msgChan = make(chan rss.ParserMsg)
		done    = make(chan struct{})
	)

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	flag.StringVar(&configPath, "config", "cmd/server/config.toml", "Path to TOML config file")
	flag.BoolVar(&dev, "dev", false, "Run the server in development mode with in-memory DB.")
	flag.StringVar(&httpAddr, "http", ":8066", "HTTP server address in the form 'host:port'.")
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

	switch dev {
	case false:
		conf := postgres.Config{
			User:     "postgres",
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			DBName:   "news",
		}
		if !conf.IsValid() {
			log.Fatal(fmt.Errorf("[server] invalid postgres config: %+v", conf))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		db, err := postgres.New(ctx, conf.ConString())
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		err = db.Ping(ctx)
		if err != nil {
			log.Fatal(fmt.Errorf("%w: %v", storage.ErrDBNotResponding, err))
		}
		log.Infof("[server] connected to postgres: %s", conf)
		sdb = db

	case true:
		log.Info("[server] running with in memory DB")
		sdb = memdb.New()
	}

	conf, err := rss.LoadConf("cmd/server/config.json")
	if err != nil {
		log.Fatalf("[server] unable to load RSS parser config: %v", err)
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

	api := api.New(cfg.ServiceName, sdb, kafkaWriter)
	parser := rss.NewParser(*conf)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer func() {
			log.Infof("[server] message receiver stopped")
			wg.Done()
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for msg := range msgChan {
			if msg.Err != nil {
				log.Warnf("[server] error while parsing %s: %v", msg.Source, msg.Err)
			} else {
				err := api.DB.AddPosts(ctx, storage.ValidatePosts(msg.Data...))
				if err != nil {
					log.Warnf("[server] error while adding posts from %s to DB: %v", msg.Source, err)
				} else {
					log.Infof("[server] DB updated with posts from %s", msg.Source)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		ticker := time.NewTicker(parser.Delay)

		defer func() {
			close(msgChan)
			ticker.Stop()
			log.Info("[server] parser stopped")
			wg.Done()
		}()

		parser.Run(msgChan)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				parser.Run(msgChan)
			}
		}
	}()

	server := &http.Server{
		Addr:    httpAddr,
		Handler: api.Router,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[server] HTTP server error: %v", err)
		}
		log.Info("[server] stopped serving new connections")
	}()

	<-sigChan
	close(done)
	wg.Wait()

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[server] HTTP shutdown error: %v", err)
	}
	log.Info("[server] stopped")
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
