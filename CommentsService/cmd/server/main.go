package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"comments/pkg/api"
	"comments/pkg/mongo"
)

const serverPort = ":8077"

func main() {
	conf, err := mongo.NewConfig()
	if err != nil {
		log.Errorf("[server] failed to connect to Mongo: %v", err)
		return
	}

	db, err := mongo.New(conf)
	if err != nil {
		log.Errorf("[server] failed to initialize storage instance, DB connection not established: %v", err)
		return
	}

	api := api.New(db)
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

	db.Close(shutdownCtx)
	log.Info("[server] disconnected from DB")
}
