package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"comments/pkg/models"
	"comments/pkg/mongo"
)

type API struct {
	ServiceName string

	r  *mux.Router
	db *mongo.Storage
	kw *kafka.Writer
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.Use(api.requestIDMiddleware)
	api.r.Use(api.headerMiddleware)

	if api.kw != nil {
		api.r.Use(api.loggingMiddleware(api.kw))
	}

	api.r.HandleFunc("/comments", api.createCommentHandler).Methods(http.MethodPost)
	api.r.HandleFunc("/comments", api.commentsHandler).
		Queries("post_id", "{[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}").
		Methods(http.MethodGet)
}

func New(name string, db *mongo.Storage, kw *kafka.Writer) *API {
	api := API{ServiceName: name, r: mux.NewRouter(), db: db}
	api.endpoints()

	return &api
}

func (api *API) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	var comment models.Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Errorf("[createCommentHandler][%s] failed to decode request body: %v", sID, err)
		return
	}
	defer r.Body.Close()

	comment, err = api.db.CreateComment(r.Context(), comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Errorf("[createCommentHandler][%s] failed to create comment: %v", sID, err)
		return
	}

	err = json.NewEncoder(w).Encode(comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Errorf("[createCommentHandler][%s] failed to encode comment: %v", sID, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	log.Debugf("[createCommentHandler][%s] comment created", sID)
}

func (api *API) commentsHandler(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	postIDStr := r.URL.Query().Get("post_id")
	if postIDStr == "" {
		http.Error(w, "Missing post_id query parameter", http.StatusBadRequest)
		log.Debugf("[commentsHandler][%s] request with empty post_id parameter", sID)
		return
	}

	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post_id format", http.StatusBadRequest)
		log.Debugf("[commentsHandler][%s] failed to parse post ID: %v", sID, err)
		return
	}

	comments, err := api.db.Comments(r.Context(), postID)
	if err != nil {
		if errors.Is(err, mongo.ErrCommentsNotFound) {
			http.Error(w, "Comments not found", http.StatusNotFound)
			log.Debugf("[commentsHandler][%s] failed to retrieve comments: %v", sID, err)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Errorf("[commentsHandler][%s] failed to retrieve comments: %v", sID, err)
		return
	}

	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Errorf("[commentsHandler][%s] failed to encode response: %v", sID, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Debugf("[commentsHandler][%s] comments retrieved", sID)
}

// GetRequestID extracts the request ID from the context.
// It returns the request ID as a string if present, otherwise returns an empty string.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

// shorten truncates a string to 6 characters if it is longer than 6, appends '...' at the end,
// otherwise it returns the string unchanged.
func shorten(s string) string {
	if len(s) > 6 {
		return s[:6] + "..."
	}
	return s
}
