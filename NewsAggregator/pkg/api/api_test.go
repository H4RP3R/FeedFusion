package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"

	"news/pkg/storage"
	"news/pkg/storage/memdb"
)

const testPostsPath = "../../test_data/post_examples.json"

func TestMain(m *testing.M) {
	log.SetLevel(log.PanicLevel)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestAPI_postsHandler(t *testing.T) {
	db := memdb.New()

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatalf("unexpected error while loading test posts: %v", err)
	}

	err = db.AddPosts(testPosts)
	if err != nil {
		t.Fatalf("unexpected error while adding posts: %v", err)
	}

	api := New(db)
	path := fmt.Sprintf("/news/%d", len(testPosts))
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()

	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got status code %v", http.StatusOK, rr.Code)
	}

	b, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("unexpected error while reading response body: %v", err)
	}

	var posts []storage.Post
	err = json.Unmarshal(b, &posts)
	if err != nil {
		t.Errorf("unexpected error while unmarshaling posts data: %v", err)
	}

	wantPosts := len(testPosts)
	gotPosts := len(posts)
	if wantPosts != gotPosts {
		t.Errorf("want %d posts, got %d posts", wantPosts, gotPosts)
	}
}

func TestAPI_postsHandlerInvalidNewsNum(t *testing.T) {
	tests := []struct {
		name       string
		n          int
		statusWant int
	}{
		{name: "n > 1000", n: 1001, statusWant: http.StatusBadRequest},
		{name: "n = 0", n: 0, statusWant: http.StatusBadRequest},
		{name: "negative n", n: -1, statusWant: http.StatusBadRequest},
		{name: "valid n", n: 1000, statusWant: http.StatusOK},
	}

	db := memdb.New()
	api := New(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/news/%d", tt.n)
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()

			api.Router.ServeHTTP(rr, req)
			if rr.Code != tt.statusWant {
				t.Errorf("want status code %v, got status code %v", tt.statusWant, rr.Code)
			}
		})
	}

	// Test none-integer n.
	t.Run("string n", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/news/abc", nil)
		rr := httptest.NewRecorder()
		api.Router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("want status code %v, got status code %v", http.StatusBadRequest, rr.Code)
		}
	})

	// Test float n.
	t.Run("float n", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/news/3.14", nil)
		rr := httptest.NewRecorder()
		api.Router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("want status code %v, got status code %v", http.StatusBadRequest, rr.Code)
		}
	})
}
