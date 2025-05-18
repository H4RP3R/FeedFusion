package api

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"news/pkg/storage"
)

const maxPostsLimit = 100

type API struct {
	ServiceName string
	DB          storage.Storage
	Router      *mux.Router
	kw          *kafka.Writer
}

func New(name string, db storage.Storage, kafkaWriter *kafka.Writer) *API {
	api := API{
		ServiceName: name,
		DB:          db,
		Router:      mux.NewRouter(),
		kw:          kafkaWriter,
	}
	api.endpoints()

	return &api
}

func (api *API) endpoints() {
	api.Router.Use(api.requestIDMiddleware)
	api.Router.Use(api.headerMiddleware)

	if api.kw != nil {
		api.Router.Use(api.loggingMiddleware(api.kw))
	}

	api.Router.HandleFunc("/news/filter", api.filterPostsHandler).Methods(http.MethodGet)
	api.Router.HandleFunc("/news/latest", api.latestPostsHandler).Methods(http.MethodGet)
	api.Router.HandleFunc("/news/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$}", api.postDetailedHandler).Methods(http.MethodGet)
}

func (api *API) latestPostsHandler(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}

	if limit > maxPostsLimit {
		http.Error(w, "Limit parameter is too big", http.StatusBadRequest)
		log.Debugf("[latestPostsHandler][%s] request with too big limit parameter", sID)
		return
	}

	posts, numPages, err := api.DB.LatestPosts(r.Context(), page, limit)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[latestPostsHandler][%s] LatestPosts() returned error: %v", sID, err)
		return
	}

	resp := PostsResponse{
		Posts:      posts,
		Pagination: Pagination{TotalPages: numPages, CurrentPage: page, Limit: limit},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[latestPostsHandler][%s] failed to encode response data: %v", sID, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Debugf("[latestPostsHandler][%s] response sent to: %v", sID, r.RemoteAddr)
}

func (api *API) filterPostsHandler(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	contains := r.URL.Query().Get("contains")
	if contains == "" {
		http.Error(w, "Empty contains parameter", http.StatusBadRequest)
		log.Debugf("[filterPostsHandler][%s] request with empty contains parameter", sID)
		return
	}
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > maxPostsLimit {
		http.Error(w, "Limit parameter is too big", http.StatusBadRequest)
		log.Debugf("[filterPostsHandler][%s] request with too big limit parameter", sID)
		return
	}

	posts, numPages, err := api.DB.FilterPosts(r.Context(), contains, page, limit)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[filterPostsHandler][%s] FilterPosts() returned error: %v", sID, err)
		return
	}

	resp := PostsResponse{
		Posts:      posts,
		Pagination: Pagination{TotalPages: numPages, CurrentPage: page, Limit: limit},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[filterPostsHandler][%s] failed to encode response data: %v", sID, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Debugf("[filterPostsHandler][%s] response sent to: %v", sID, r.RemoteAddr)
}

func (api *API) postDetailedHandler(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	idStr := mux.Vars(r)["id"]
	id, err := uuid.FromString(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID parameter", http.StatusBadRequest)
		log.Debugf("[postDetailedHandler][%s] failed to parse post ID: %v", sID, err)
		return
	}

	post, err := api.DB.Post(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrPostNotFound) {
			http.Error(w, "Post not found", http.StatusNotFound)
			log.Debugf("[postDetailedHandler][%s] failed to retrieve post: %v", sID, err)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postDetailedHandler][%s] post ID:%v: %v", sID, id, err)
		return
	}

	err = json.NewEncoder(w).Encode(post)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postDetailedHandler][%s] failed to encode post data: %v", sID, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Debugf("[postDetailedHandler][%s] response sent to: %v", sID, r.RemoteAddr)
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
