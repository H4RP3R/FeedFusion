package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"gateway/pkg/models"
)

const (
	commentsServiceURL = "http://localhost:8077"
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

	api.r.HandleFunc("/news/latest", api.latestNewsProxy).Queries("page", "{page:[0-9]+}").Methods(http.MethodGet)
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
	_ = r.URL.Query().Get("page")
	var previews []models.Preview
	for _, n := range mockNews {
		var prev models.Preview
		prev.ID = n.ID
		prev.Title = n.Title
		prev.Link = n.Link
		prev.Published = n.Published
		previews = append(previews, prev)
	}

	err := json.NewEncoder(w).Encode(previews)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// curl 'http://localhost:8080/news/filter?startDate=2023-01-01&endDate=2023-04-01&contains=text&sortField=date&sortDirection=asc'
func (api *API) filterNewsProxy(w http.ResponseWriter, r *http.Request) {
	_ = r.URL.Query().Get("startDate")
	_ = r.URL.Query().Get("endDate")
	_ = r.URL.Query().Get("contains")
	_ = r.URL.Query().Get("sortField")
	_ = r.URL.Query().Get("sortDirection")

	var previews []models.Preview
	for _, n := range mockNews {
		var prev models.Preview
		prev.ID = n.ID
		prev.Title = n.Title
		prev.Link = n.Link
		prev.Published = n.Published
		previews = append(previews, prev)
	}

	err := json.NewEncoder(w).Encode(previews)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) newsDetailedProxy(w http.ResponseWriter, r *http.Request) {
	idStr, ok := mux.Vars(r)["id"]
	id, err := uuid.FromString(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	}

	for _, post := range mockNews {
		if post.ID == id {
			json.NewEncoder(w).Encode(post)
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
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

var mockNews = []models.Post{
	{
		ID:    uuid.FromStringOrNil("1c0bbc26-70d1-5af4-9785-92bd490a3075"),
		Title: "Goroutines in Go: Lightweight Concurrency",
		Content: `Goroutines are a fundamental feature in the Go programming language that enable lightweight concurrency. They allow developers to write efficient and scalable concurrent programs with ease
		A goroutine is a function or method that runs concurrently with other functions or methods. It's a separate unit of execution that can run in parallel with other goroutines. Goroutines are scheduled and managed by the Go runtime, which handles the complexity of concurrency for you.`,
		Published: time.Date(2025, 9, 28, 0, 0, 0, 0, time.UTC),
		Link:      "https://tech/posts/1234",
	},
	{
		ID:    uuid.FromStringOrNil("3505605d-861f-591e-a654-e95e9d83cc7e"),
		Title: "Classes in Python: A Guide to Object-Oriented Programming",
		Content: `In Python, classes are a fundamental concept in object-oriented programming (OOP). They allow you to define custom data types and behaviors, enabling you to write more organized, reusable, and maintainable code.
		A class is a blueprint or template that defines the properties and behaviors of an object. It's a way to define a custom data type that can have its own attributes (data) and methods (functions).`,
		Published: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC),
		Link:      "https://tech/posts/1010",
	},
	{
		ID:        uuid.FromStringOrNil("f3767624-65e9-5e26-80e1-aea970710389"),
		Title:     "The Rise of AI Code Assistants: Revolutionizing Software Development",
		Content:   `The world of software development is undergoing a significant transformation with the emergence of AI code assistants. These intelligent tools are designed to assist developers in writing, debugging, and optimizing their code, making the development process faster, more efficient, and more enjoyable.`,
		Published: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		Link:      "https://tech/posts/1198",
	},
}
