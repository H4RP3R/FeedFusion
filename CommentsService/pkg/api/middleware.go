package api

import (
	"context"
	"net/http"

	log "github.com/sirupsen/logrus"
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
