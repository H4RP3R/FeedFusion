package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"comments/pkg/logger"
)

type ctxKeyRequestID struct{}

var RequestIDKey = ctxKeyRequestID{}

func (api *API) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			log.Warnf("[requestIDMiddleware] missing X-Request-Id header from %v", r.RemoteAddr)
			http.Error(w, "Missing X-Request-Id header", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (api *API) headerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (api *API) loggingMiddleware(kWriter *kafka.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lw := logger.New(w)
			defer func() {
				go func() {
					entry := LogEntry{
						Timestamp:  time.Now(),
						IP:         getClientIP(r),
						StatusCode: lw.Status(),
						RequestID:  GetRequestID(r.Context()),
						Method:     r.Method,
						Path:       r.URL.Path,
						Duration:   time.Since(start).Seconds(),
						Service:    api.ServiceName,
					}

					jsonEntry, err := json.Marshal(entry)
					if err != nil {
						log.Errorf("[LoggingMiddleware] failed to marshal log entry for request %s", entry.RequestID)
						return
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					err = kWriter.WriteMessages(ctx, kafka.Message{Value: jsonEntry})
					if err != nil {
						log.Errorf("[LoggingMiddleware] failed to write log to Kafka: %v", err)
						return
					}
					log.Debugf("[LoggingMiddleware] log entry sent to Kafka request_id:%s", entry.RequestID)
				}()
			}()

			next.ServeHTTP(lw, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}

	return ip
}
