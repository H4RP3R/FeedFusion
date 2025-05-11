package api

import (
	"encoding/json"
	"fmt"

	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"news/pkg/storage"
)

const maxPosts = 1000

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
	api.Router.HandleFunc("/news/{n}", api.postsHandler).Methods(http.MethodGet)
}

// postsHandler handles GET requests to /news/{n} and returns n latest posts from
// the underlying storage in JSON format.
func (api *API) postsHandler(w http.ResponseWriter, r *http.Request) {
	nStr := mux.Vars(r)["n"]
	n, err := strconv.Atoi(nStr)
	if err != nil {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		log.Infof("[postsHandler] from %v: %v", r.RemoteAddr, err)
		return
	}

	if n < 1 || n > 1000 {
		http.Error(w, fmt.Sprintf("Invalid news num (1 <= n <= %d)", maxPosts), http.StatusBadRequest)
		log.Infof("[postsHandler] from %v: %v", r.RemoteAddr, "Invalid news num")
		return
	}

	posts, err := api.DB.Posts(n)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postsHandler] status %v: %v", http.StatusInternalServerError, err)
		return
	}

	err = json.NewEncoder(w).Encode(posts)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		log.Errorf("[postsHandler] status %v: %v", http.StatusInternalServerError, err)
		return
	}

	log.Infof("[postsHandler] response sent to %v", r.RemoteAddr)
}
