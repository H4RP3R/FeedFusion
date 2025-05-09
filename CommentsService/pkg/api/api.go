package api

import (
	"encoding/json"
	"net/http"

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
	api.r.HandleFunc("/comments", api.createCommentHandler).Methods(http.MethodPost)
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

	comment, err = api.db.CreateComment(comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}
