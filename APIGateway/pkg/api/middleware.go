package api

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

type ctxKeyRequestID struct{}

var RequestIDKey = ctxKeyRequestID{}

func (api *API) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			id, err := uuid.NewV4()
			if err != nil {
				log.Errorf("[requestIDMiddleware] failed to generate request ID for %v: %v", r.RemoteAddr, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			reqID = id.String()
			log.Debugf("[requestIDMiddleware] generated request ID:%s for %v", reqID, r.RemoteAddr)
		}

		w.Header().Set("X-Request-Id", reqID)
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
