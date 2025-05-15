package main

import (
	"context"
	"errors"
	"gateway/pkg/api"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const serverPort = ":8088"

func main() {
	api, err := api.New("pkg/api/config.json")
	if err != nil {
		log.Fatalf("[server] failed to create API: %v", err)
	}

	srv := &http.Server{
		Addr:    serverPort,
		Handler: api.Router(),
	}

	go func() {
		log.Infof("[server] starting on port %v", serverPort)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("[server] failed to start: %v", err)
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
