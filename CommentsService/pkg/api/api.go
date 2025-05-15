package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"

	"comments/pkg/models"
	"comments/pkg/mongo"
)

type API struct {
	r  *mux.Router
	db *mongo.Storage
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.Use(api.headerMiddleware)

	api.r.HandleFunc("/comments", api.createCommentHandler).Methods(http.MethodPost)
	api.r.HandleFunc("/comments", api.commentsHandler).
		Queries("post_id", "{[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}").
		Methods(http.MethodGet)
}

func New(db *mongo.Storage) *API {
	api := API{r: mux.NewRouter(), db: db}
	api.endpoints()

	return &api
}

func (api *API) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	var comment models.Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	comment, err = api.db.CreateComment(r.Context(), comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func (api *API) commentsHandler(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.URL.Query().Get("post_id")
	if postIDStr == "" {
		http.Error(w, "Missing post_id query parameter", http.StatusBadRequest)
		return
	}

	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post_id format", http.StatusBadRequest)
		return
	}

	comments, err := api.db.Comments(r.Context(), postID)
	if err != nil {
		if errors.Is(err, mongo.ErrCommentsNotFound) {
			http.Error(w, "Comments not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
