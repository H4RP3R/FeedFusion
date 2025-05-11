package memdb

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"news/pkg/storage"
)

const testPostsPath = "../../../test_data/post_examples.json"

func TestDB_AddPost(t *testing.T) {
	db := New()

	testPosts, err := LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	for i, post := range testPosts {
		testPosts[i].ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
	}

	for _, post := range testPosts {
		gotID, err := db.AddPost(post)
		if err != nil {
			t.Errorf("unexpected error while adding post: %v", err)
		}
		if gotID != post.ID {
			t.Errorf("want post ID %v, got post ID %v", post.ID, gotID)
		}
	}

	if len(db.posts) != len(testPosts) {
		t.Errorf("want posts in DB %d, got posts in DB %d", len(testPosts), len(db.posts))
	}
}

func TestDB_AddPosts(t *testing.T) {
	db := New()

	testPosts, err := LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	err = db.AddPosts(testPosts)
	if err != nil {
		t.Errorf("unexpected error while adding posts: %v", err)
	}
	if len(db.posts) != len(testPosts) {
		t.Errorf("want posts count %d, got posts count %d", len(testPosts), len(db.posts))
	}
}

func TestDB_Posts(t *testing.T) {
	db := New()

	// Test posts from newest to oldest.
	testPosts := []storage.Post{
		{Title: "Seventh Post", Content: "Content 7", Published: time.Date(2025, 9, 28, 0, 0, 0, 0, time.UTC), Link: "https://example.com/7"},
		{Title: "Sixth Post", Content: "Content 6", Published: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), Link: "https://example.com/6"},
		{Title: "Fifth Post", Content: "Content 5", Published: time.Date(2025, 8, 0, 0, 0, 0, 0, time.UTC), Link: "https://example.com/5"},
		{Title: "Fourth Post", Content: "Content 4", Published: time.Date(2025, 3, 13, 5, 0, 15, 0, time.UTC), Link: "https://example.com/4"},
		{Title: "Third Post", Content: "Content 3", Published: time.Date(2025, 3, 13, 5, 0, 10, 0, time.UTC), Link: "https://example.com/3"},
		{Title: "Second Post", Content: "Content 2", Published: time.Date(2024, 10, 8, 22, 2, 0, 0, time.UTC), Link: "https://example.com/2"},
		{Title: "First Post", Content: "Content 1", Published: time.Date(2024, 10, 8, 22, 0, 0, 0, time.UTC), Link: "https://example.com/1"},
	}

	var err error
	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(post)
		if err != nil {
			t.Fatalf("unexpected error while adding posts: %v", err)
		}
	}

	tests := []struct {
		n       int
		wantCnt int
	}{
		{n: 0, wantCnt: 0},
		{n: 1, wantCnt: 1},
		{n: 5, wantCnt: 5},
		{n: 6, wantCnt: 6},
		{n: 7, wantCnt: 7},
		{n: 8, wantCnt: 7},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("n=%d, posts in DB %d", tt.n, len(testPosts))
		t.Run(testName, func(t *testing.T) {
			gotPosts, err := db.Posts(tt.n)
			if err != nil {
				t.Errorf("unexpected error while getting posts: %v", err)
			}
			if len(gotPosts) != tt.wantCnt {
				t.Errorf("want posts in response %d, got posts in response %d", tt.wantCnt, len(gotPosts))
			}
			if !reflect.DeepEqual(testPosts[:tt.wantCnt], gotPosts) {
				t.Errorf("want posts \n%+v\ngot posts\n%+v\n", testPosts[:tt.wantCnt], gotPosts)
			}
		})
	}
}
