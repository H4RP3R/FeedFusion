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

	"gateway/pkg/api"
)

func main() {
	var (
		httpAddr string
		logLevel string
	)

	flag.StringVar(&httpAddr, "http", ":8088", "HTTP server address in the form 'host:port'.")
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

	api, err := api.New("pkg/api/config.json")
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
