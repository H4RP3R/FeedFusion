package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
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

func TestAPI_latestPostsHandler(t *testing.T) {
	db := memdb.New()

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatalf("unexpected error while loading test posts: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = db.AddPosts(ctx, testPosts)
	if err != nil {
		t.Fatalf("unexpected error while adding posts: %v", err)
	}

	api := New(db)
	path := fmt.Sprintf("/news/latest?page=1&limit=%d", len(testPosts))
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

	var resp PostsResponse
	err = json.Unmarshal(b, &resp)
	if err != nil {
		t.Errorf("unexpected error while unmarshaling response data: %v", err)
	}

	wantPosts := len(testPosts)
	gotPosts := len(resp.Posts)
	if wantPosts != gotPosts {
		t.Errorf("want %d posts, got %d posts", wantPosts, gotPosts)
	}
}

func TestAPI_postDetailedHandlerLimitExceeded(t *testing.T) {
	db := memdb.New()
	api := New(db)

	// Expect 400
	path := fmt.Sprintf("/news/latest?page=1&limit=%d", maxPostsLimit+1)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("want status code %v, got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestAPI_filterPostsHandler(t *testing.T) {
	db := memdb.New()
	api := New(db)

	// Expect 200
	req := httptest.NewRequest(http.MethodGet, "/news/filter?contains=some_text", nil)
	rr := httptest.NewRecorder()
	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got %v", http.StatusOK, rr.Code)
	}

	// Expect 400
	req = httptest.NewRequest(http.MethodGet, "/news/filter?contains=", nil)
	rr = httptest.NewRecorder()
	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("want status code %v, got %v", http.StatusBadRequest, rr.Code)
	}

	// Expect 400
	path := fmt.Sprintf("/news/filter?contains=some_text&page=1&limit=%d", maxPostsLimit+1)
	req = httptest.NewRequest(http.MethodGet, path, nil)
	rr = httptest.NewRecorder()
	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("want status code %v, got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestAPI_postDetailedHandler(t *testing.T) {
	db := memdb.New()
	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatalf("unexpected error while loading test posts: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = db.AddPosts(ctx, testPosts)
	if err != nil {
		t.Fatalf("unexpected error while adding posts: %v", err)
	}

	api := New(db)
	targetPost := testPosts[0]
	targetPostID := targetPost.ID
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/news/%s", targetPostID.String()), nil)
	rr := httptest.NewRecorder()

	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got status code %v", http.StatusOK, rr.Code)
	}

	b, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("unexpected error while reading response body: %v", err)
	}

	var gotPost storage.Post
	err = json.Unmarshal(b, &gotPost)
	if err != nil {
		t.Errorf("unexpected error while unmarshaling post data: %v", err)
	}

	if !reflect.DeepEqual(targetPost, gotPost) {
		t.Errorf("want post\n%+v\n\ngot post\n%+v\n", targetPost, gotPost)
	}
}

func TestAPI_postDetailedHandlerNotExist(t *testing.T) {
	db := memdb.New()
	api := New(db)

	targetPostID, err := uuid.FromString("01234567-89ab-cdef-0123-456789abcdef")
	if err != nil {
		t.Fatalf("unexpected error while parsing UUID: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/news/%s", targetPostID.String()), nil)
	rr := httptest.NewRecorder()

	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("want status code %v, got status code %v", http.StatusNotFound, rr.Code)
	}
}

func TestAPI_postDetailedHandlerInvalidUUID(t *testing.T) {
	db := memdb.New()
	api := New(db)

	req := httptest.NewRequest(http.MethodGet, "/news/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	api.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("want status code %v, got status code %v", http.StatusNotFound, rr.Code)
	}
}
