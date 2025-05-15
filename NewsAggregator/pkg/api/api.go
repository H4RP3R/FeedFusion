package api

import (
	"encoding/json"
	"errors"
	"strconv"

	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"news/pkg/storage"
)

const maxPostsLimit = 100

type API struct {
	DB     storage.Storage
	Router *mux.Router
}

func New(db storage.Storage) *API {
	api := API{
		DB:     db,
		Router: mux.NewRouter(),
	}
	api.endpoints()

	return &api
}

func (api *API) endpoints() {
	api.Router.Use(api.headerMiddleware)
	api.Router.HandleFunc("/news/filter", api.filterPostsHandler).Methods(http.MethodGet)
	api.Router.HandleFunc("/news/latest", api.latestPostsHandler).Methods(http.MethodGet)
	api.Router.HandleFunc("/news/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$}", api.postDetailedHandler).Methods(http.MethodGet)
}

func (api *API) latestPostsHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Debugf("[postsHandler] request with too big limit parameter from: %v", r.RemoteAddr)
		return
	}

	posts, numPages, err := api.DB.LatestPosts(page, limit)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postsHandler] LatestPosts() returned error: %v", err)
		return
	}

	resp := PostsResponse{
		Posts:      posts,
		Pagination: Pagination{TotalPages: numPages, CurrentPage: page, Limit: limit},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postsHandler] failed to encode response data: %v", err)
		return
	}

	log.Debugf("[postsHandler] response sent to: %v", r.RemoteAddr)
}

func (api *API) filterPostsHandler(w http.ResponseWriter, r *http.Request) {
	contains := r.URL.Query().Get("contains")
	if contains == "" {
		http.Error(w, "Empty contains parameter", http.StatusBadRequest)
		log.Debugf("[filterPostsHandler] request with empty parameter from: %v", r.RemoteAddr)
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
		log.Debugf("[filterPostsHandler] request with too big limit parameter from: %v", r.RemoteAddr)
		return
	}

	posts, numPages, err := api.DB.FilterPosts(contains, page, limit)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[filterPostsHandler] FilterPosts() returned error: %v", err)
		return
	}

	resp := PostsResponse{
		Posts:      posts,
		Pagination: Pagination{TotalPages: numPages, CurrentPage: page, Limit: limit},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postsHandler] failed to encode response data: %v", err)
		return
	}

	log.Debugf("[filterPostsHandler] response sent to: %v", r.RemoteAddr)
}

func (api *API) postDetailedHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.FromString(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID parameter", http.StatusBadRequest)
		log.Debugf("[postDetailedHandler] from %v: %v", r.RemoteAddr, err)
		return
	}

	post, err := api.DB.Post(id)
	if err != nil {
		if errors.Is(err, storage.ErrPostNotFound) {
			http.Error(w, "Post not found", http.StatusNotFound)
			log.Debugf("[postDetailedHandler] from %v: %v", r.RemoteAddr, err)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postDetailedHandler] failed to retrieve post: %v", err)
		return
	}

	err = json.NewEncoder(w).Encode(post)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postDetailedHandler] failed to encode post data: %v", err)
		return
	}

	log.Debugf("[postDetailedHandler] response sent to: %v", r.RemoteAddr)
}
