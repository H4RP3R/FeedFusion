package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"censor/pkg/models"
)

type API struct {
	ServiceName string

	r  *mux.Router
	kw *kafka.Writer
}

func New(name string, kafkaWriter *kafka.Writer) (*API, error) {
	api := API{
		ServiceName: name,
		r:           mux.NewRouter(),
		kw:          kafkaWriter,
	}
	api.endpoints()

	return &api, nil
}

func (api *API) Router() *mux.Router {
	return api.r
}
func (api *API) endpoints() {
	api.r.Use(api.requestIDMiddleware)
	api.r.Use(api.headerMiddleware)

	api.r.HandleFunc("/check", api.checkComment).Methods(http.MethodPost)

	if api.kw != nil {
		api.r.Use(api.loggingMiddleware(api.kw))
	}
}

func (apo *API) checkComment(w http.ResponseWriter, r *http.Request) {
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

	// ! For testing purposes
	fmt.Printf("%+v\n", comment)
	w.WriteHeader(http.StatusOK)
}

// shorten truncates a string to 6 characters if it is longer than 6, appends '...' at the end,
// otherwise it returns the string unchanged.
func shorten(s string) string {
	if len(s) > 6 {
		return s[:6] + "..."
	}
	return s
}
