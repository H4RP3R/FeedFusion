package api

import (
	"bytes"
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
	log "github.com/sirupsen/logrus"

	"gateway/pkg/models"
)

const (
	commentsServiceURL = "http://localhost:8077"
	newsServiceURL     = "http://localhost:8066"
	timeout            = 5 * time.Second
)

type API struct {
	r *mux.Router
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.Use(api.headerMiddleware)

	api.r.HandleFunc("/news/latest", api.latestNewsProxy).Methods(http.MethodGet)
	api.r.HandleFunc("/news/filter", api.filterNewsProxy).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$}", api.newsDetailedProxy).Methods(http.MethodGet)

	api.r.HandleFunc("/comments", api.createCommentProxy).Methods(http.MethodPost)
}

func New() *API {
	api := API{r: mux.NewRouter()}
	api.endpoints()

	return &api
}

func (api *API) latestNewsProxy(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	itemsPerPageStr := r.URL.Query().Get("limit")
	itemsPerPage := 10
	if itemsPerPageStr != "" {
		if p, err := strconv.Atoi(itemsPerPageStr); err == nil && p > 0 {
			itemsPerPage = p
		}
	}
	if itemsPerPage > 100 {
		itemsPerPage = 100
	}

	targetURL := fmt.Sprintf("%s/news/latest?page=%d&limit=%d", newsServiceURL, page, itemsPerPage)

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, nil)
	if err != nil {
		log.Errorf("[latestNewsProxy][from:%v] error creating proxy request %s %s: %v", r.RemoteAddr, r.Method, targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header.Clone()
	// Remove hop-by-hop headers
	proxyReq.Header.Del("Connection")
	proxyReq.Header.Del("Keep-Alive")
	proxyReq.Header.Del("Proxy-Authenticate")
	proxyReq.Header.Del("Proxy-Authorization")
	proxyReq.Header.Del("TE")
	proxyReq.Header.Del("Trailer")
	proxyReq.Header.Del("Transfer-Encoding")
	proxyReq.Header.Del("Upgrade")

	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("[latestNewsProxy][from:%v] error proxying request %s %s: %v", r.RemoteAddr, r.Method, targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("[latestNewsProxy][from:%v] error copying response body: %v", r.RemoteAddr, err)
	}
}

func (api *API) filterNewsProxy(w http.ResponseWriter, r *http.Request) {
	contains := r.URL.Query().Get("contains")
	if contains == "" {
		log.Debugf("[filterNewsProxy][from:%v] empty contains parameter", r.RemoteAddr)
		http.Error(w, "Empty contains parameter", http.StatusBadRequest)
		return
	}

	targetURL := fmt.Sprintf("%s/news/filter?contains=%s", newsServiceURL, url.QueryEscape(contains))

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, nil)
	if err != nil {
		log.Errorf("[filterNewsProxy][from:%v] error creating proxy request: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = cloneHeaderNoHop(r.Header)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("[filterNewsProxy][from:%v] error calling news aggregator: %v", r.RemoteAddr, err)
		http.Error(w, "News Aggregator Unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("[filterNewsProxy][from:%v] error copying response body: %v", r.RemoteAddr, err)
	}
}

func (api *API) newsDetailedProxy(w http.ResponseWriter, r *http.Request) {
	idStr, ok := mux.Vars(r)["id"]
	if !ok {
		log.Debugf("[newsDetailedProxy][from:%v] missing id parameter", r.RemoteAddr)
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

		url, _ := url.Parse(commentsServiceURL)
		url = url.JoinPath("comments")
		values := url.Query()
		values.Set("post_id", idStr)
		url.RawQuery = values.Encode()
		fetchResource(client, url.String(), "comments service", &[]models.Comment{}, respChan)

	}(wg, client)

	// News sub request
	go func(wg *sync.WaitGroup, client *http.Client) {
		defer wg.Done()

		url, _ := url.Parse(newsServiceURL)
		url = url.JoinPath(url.Path, "news", idStr)
		fetchResource(client, url.String(), "news aggregator", &models.Post{}, respChan)
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
			var errNotFound *ErrNotFound
			if errors.As(v, &errNotFound) {
				http.Error(w, "Post not found", http.StatusNotFound)
				log.Infof("[newsDetailedProxy][from:%v] %v", r.RemoteAddr, errNotFound)
				return
			}
			log.Errorf("[newsDetailedProxy][from:%v] error in sub request: %v", r.RemoteAddr, msg)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	post.Comments = comments

	if err := json.NewEncoder(w).Encode(post); err != nil {
		log.Errorf("[newsDetailedProxy][from:%v] error encoding response: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (api *API) createCommentProxy(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("[createCommentProxy][from:%v] error reading request body: %v", r.RemoteAddr, err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Cut off invalid requests
	var comment models.Comment
	if err := json.Unmarshal(b, &comment); err != nil {
		log.Errorf("[createCommentProxy][from:%v] invalid JSON: %v", r.RemoteAddr, err)
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}
	if comment.PostID == uuid.Nil && comment.ParentID == uuid.Nil {
		log.Errorf("[createCommentProxy][from:%v] missing post_id and parent_id", r.RemoteAddr)
		http.Error(w, "Bad Request: missing post_id or parent_id", http.StatusBadRequest)
		return
	}

	targetURL := fmt.Sprint(commentsServiceURL + "/comments")

	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(b))
	if err != nil {
		log.Errorf("[createCommentProxy][from:%v] error creating proxy request: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = cloneHeaderNoHop(r.Header)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("[createCommentProxy][from:%v] error calling comments service: %v", r.RemoteAddr, err)
		http.Error(w, "Comments Service Unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("[createCommentProxy][from:%v] error copying response body: %v", r.RemoteAddr, err)
	}
}

func fetchResource(client *http.Client, url, service string, resultObj any, respChan chan any) {
	proxyReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		respChan <- fmt.Errorf("error creating request to comments service: %w", err)
		return
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		respChan <- fmt.Errorf("error calling comments service: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		respChan <- &ErrNotFound{msg: service + " sub request returned 404"}
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
