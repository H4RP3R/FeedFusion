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

	log "github.com/sirupsen/logrus"

	"comments/pkg/api"
	"comments/pkg/mongo"
)

func main() {
	var (
		httpAddr string
		logLevel string
	)

	flag.StringVar(&httpAddr, "http", ":8077", "HTTP server address in the form 'host:port'.")
	flag.StringVar(&logLevel, "log", "info", "Log level: debug, info, warn, error.")
	flag.Parse()

	if !strings.Contains(httpAddr, ":") {
		log.Warn("use ':' before port number, e.g. ':8080'")
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

	conf, err := mongo.NewConfig()
	if err != nil {
		log.Errorf("[server] failed to connect to Mongo: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := mongo.New(ctx, conf)
	if err != nil {
		log.Errorf("[server] failed to initialize storage instance, DB connection not established: %v", err)
		return
	}

	api := api.New(db)
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: api.Router(),
	}

	go func() {
		log.Infof("[server] starting on port %v", httpAddr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[server] failed to start: %v", err)
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

	db.Close(shutdownCtx)
	log.Info("[server] disconnected from DB")
}
