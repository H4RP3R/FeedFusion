package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"censorship/pkg/censor"
	"censorship/pkg/models"
)

type API struct {
	ServiceName string
	Censor      *censor.Censor
	r           *mux.Router
	kw          *kafka.Writer
}

func New(name string, censor *censor.Censor, kafkaWriter *kafka.Writer) (*API, error) {
	api := API{
		ServiceName: name,
		Censor:      censor,
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

func (api *API) checkComment(w http.ResponseWriter, r *http.Request) {
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

	banned := api.Censor.Check(comment.Text)
	if banned {
		w.WriteHeader(http.StatusUnprocessableEntity)
		http.Error(w, "Comment is banned", http.StatusUnprocessableEntity)
		log.Debugf("[createCommentHandler][%s] comment is banned", sID)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Debugf("[createCommentHandler][%s] comment approved", sID)
}

// shorten truncates a string to 6 characters if it is longer than 6, appends '...' at the end,
// otherwise it returns the string unchanged.
func shorten(s string) string {
	if len(s) > 6 {
		return s[:6] + "..."
	}
	return s
}
