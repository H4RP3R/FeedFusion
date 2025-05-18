package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"gateway/pkg/models"
)

const httpClientTimeout = 5 * time.Second

type Service struct {
	URL  string
	Name string
}

type API struct {
	r  *mux.Router
	kw *kafka.Writer

	Services map[string]Service
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.Use(api.requestIDMiddleware)
	api.r.Use(api.headerMiddleware)

	api.r.HandleFunc("/news/latest", api.latestNewsProxy).Methods(http.MethodGet)
	api.r.HandleFunc("/news/filter", api.filterNewsProxy).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$}", api.newsDetailedProxy).Methods(http.MethodGet)

	api.r.HandleFunc("/comments", api.createCommentProxy).Methods(http.MethodPost)
}

func New(services map[string]Service, kafkaWriter *kafka.Writer) (*API, error) {
	api := API{
		r:        mux.NewRouter(),
		kw:       kafkaWriter,
		Services: services,
	}

	api.endpoints()
	if api.kw != nil {
		api.r.Use(api.loggingMiddleware(api.kw))
	}

	return &api, nil
}

func (api *API) latestNewsProxy(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	page, limit := parsePagination(r, 100)
	targetURL := fmt.Sprintf("%s/news/latest?page=%d&limit=%d", api.Services["Aggregator"].URL, page, limit)

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, nil)
	if err != nil {
		log.Errorf("[latestNewsProxy][%s] error creating proxy request %s %s: %v", sID, r.Method, targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = cloneHeaderNoHop(r.Header)
	if reqID != "" {
		proxyReq.Header.Set("X-Request-Id", reqID)
	}

	client := &http.Client{Timeout: httpClientTimeout}

	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("[latestNewsProxy][%s] error proxying request %s %s: %v", sID, r.Method, targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Set(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("[latestNewsProxy][%s] error copying response body: %v", sID, err)
	}

	log.Debugf("[latestNewsProxy][%s] response sent to %v", sID, r.RemoteAddr)
}

func (api *API) filterNewsProxy(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	contains := r.URL.Query().Get("contains")
	if contains == "" {
		log.Debugf("[filterNewsProxy][%s] empty contains parameter", sID)
		http.Error(w, "Empty contains parameter", http.StatusBadRequest)
		return
	}

	page, limit := parsePagination(r, 100)

	targetURL := fmt.Sprintf(
		"%s/news/filter?contains=%s&page=%d&limit=%d",
		api.Services["Aggregator"].URL,
		url.QueryEscape(contains),
		page, limit)

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, nil)
	if err != nil {
		log.Errorf("[filterNewsProxy][%s] error creating proxy request: %v", sID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = cloneHeaderNoHop(r.Header)
	if reqID != "" {
		proxyReq.Header.Set("X-Request-Id", reqID)
	}

	client := &http.Client{Timeout: httpClientTimeout}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("[filterNewsProxy][%s] error calling news aggregator: %v", sID, err)
		http.Error(w, "News Aggregator Unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Set(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("[filterNewsProxy][%s] error copying response body: %v", sID, err)
	}

	log.Debugf("[filterNewsProxy][%s] response sent to %v", sID, r.RemoteAddr)
}

func (api *API) newsDetailedProxy(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	idStr, ok := mux.Vars(r)["id"]
	if !ok {
		log.Debugf("[newsDetailedProxy][%s] missing id parameter", sID)
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	numSubRequests := 2
	respChan := make(chan any, numSubRequests)
	wg := &sync.WaitGroup{}
	wg.Add(numSubRequests)
	client := &http.Client{Timeout: 10 * time.Second}

	// Comments sub request
	go func(wg *sync.WaitGroup, client *http.Client) {
		defer wg.Done()

		url, _ := url.Parse(api.Services["Comments"].URL)
		url = url.JoinPath("comments")
		values := url.Query()
		values.Set("post_id", idStr)
		url.RawQuery = values.Encode()
		fetchResource(r.Context(), client, reqID, url.String(), "comments service", &[]models.Comment{}, respChan)

	}(wg, client)

	// News sub request
	go func(wg *sync.WaitGroup, client *http.Client) {
		defer wg.Done()

		url, _ := url.Parse(api.Services["Aggregator"].URL)
		url = url.JoinPath(url.Path, "news", idStr)
		fetchResource(r.Context(), client, reqID, url.String(), "news aggregator", &models.Post{}, respChan)
	}(wg, client)

	wg.Wait()
	close(respChan)

	var post models.Post
	var comments []models.Comment

	for msg := range respChan {
		switch v := msg.(type) {
		case *[]models.Comment:
			comments = *v
		case *models.Post:
			post = *v
		case error:
			var errSubRequest *ErrSubRequest
			if errors.As(v, &errSubRequest) {
				if v.Error() == "news aggregator sub request returned 404" {
					http.Error(w, "Post not found", http.StatusNotFound)
					log.Debugf("[newsDetailedProxy][%s] %v", sID, errSubRequest)
					return
				} else if v.Error() == "comments service sub request returned 404" {
					log.Debugf("[newsDetailedProxy][%s] %v", sID, errSubRequest)
				} else {
					log.Errorf("[newsDetailedProxy][%s] error in sub request: %v", sID, errSubRequest)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
		}
	}

	post.Comments = comments

	if err := json.NewEncoder(w).Encode(post); err != nil {
		log.Errorf("[newsDetailedProxy][%s] error encoding response: %v", sID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debugf("[newsDetailedProxy][%s] response sent to %v", sID, r.RemoteAddr)
}

func (api *API) createCommentProxy(w http.ResponseWriter, r *http.Request) {
	reqID := GetRequestID(r.Context())
	sID := shorten(reqID)

	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("[createCommentProxy][%s] error reading request body: %v", sID, err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Cut off invalid requests
	var comment models.Comment
	if err := json.Unmarshal(b, &comment); err != nil {
		log.Errorf("[createCommentProxy][%s] invalid JSON: %v", sID, err)
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}
	if comment.PostID == uuid.Nil && comment.ParentID == uuid.Nil {
		log.Errorf("[createCommentProxy][%s] missing post_id and parent_id", sID)
		http.Error(w, "Bad Request: missing post_id or parent_id", http.StatusBadRequest)
		return
	}

	targetURL := fmt.Sprint(api.Services["Comments"].URL + "/comments")

	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(b))
	if err != nil {
		log.Errorf("[createCommentProxy][%s] error creating proxy request: %v", sID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = cloneHeaderNoHop(r.Header)
	if reqID != "" {
		proxyReq.Header.Set("X-Request-Id", reqID)
	}

	client := &http.Client{Timeout: httpClientTimeout}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("[createCommentProxy][%s] error calling comments service: %v", sID, err)
		http.Error(w, "Comments Service Unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Set(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("[createCommentProxy][%s] error copying response body: %v", sID, err)
	}

	log.Debugf("[createCommentProxy][%s] response sent to %v", sID, r.RemoteAddr)
}

func fetchResource(ctx context.Context, client *http.Client, reqID, url, service string, resultObj any, respChan chan any) {
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		respChan <- fmt.Errorf("error creating request to %s: %w", service, err)
		return
	}

	if reqID != "" {
		proxyReq.Header.Set("X-Request-Id", reqID)
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		respChan <- fmt.Errorf("error calling comments service: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		respChan <- &ErrSubRequest{msg: service + " sub request returned 404"}
		return
	}

	if resp.StatusCode != http.StatusOK {
		respChan <- fmt.Errorf("%s returned status %d", service, resp.StatusCode)
		return
	}

	if err := json.NewDecoder(resp.Body).Decode(&resultObj); err != nil {
		respChan <- fmt.Errorf("error decoding response from %s: %w", service, err)
		return
	}

	respChan <- resultObj
}

func cloneHeaderNoHop(header http.Header) http.Header {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	h := header.Clone()
	for _, key := range hopByHopHeaders {
		h.Del(key)
	}

	return h
}

// parsePagination extracts and validates 'page' and 'limit' query parameters from the request.
// It returns default values if parameters are missing or invalid.
// 'maxLimit' caps the maximum allowed limit to prevent abuse.
func parsePagination(r *http.Request, maxLimit int) (page, limit int) {
	page = 1
	limit = 10

	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	return
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
