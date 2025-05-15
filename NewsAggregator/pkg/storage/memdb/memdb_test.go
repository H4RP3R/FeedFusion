package memdb

import (
	"context"
	"reflect"
	"slices"
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
		gotID, err := db.AddPost(context.Background(), post)
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

	err = db.AddPosts(context.Background(), testPosts)
	if err != nil {
		t.Errorf("unexpected error while adding posts: %v", err)
	}
	if len(db.posts) != len(testPosts) {
		t.Errorf("want posts count %d, got posts count %d", len(testPosts), len(db.posts))
	}
}

func TestDB_LatestPosts(t *testing.T) {
	db := New()

	// Test posts from newest to oldest.
	testPosts := []storage.Post{
		{
			Title:     "Seventh Post",
			Content:   "Content 7",
			Published: time.Date(2025, 9, 28, 0, 0, 0, 0, time.UTC),
			Link:      "https://example.com/7",
		},
		{
			Title:     "Sixth Post",
			Content:   "Content 6",
			Published: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
			Link:      "https://example.com/6",
		},
		{
			Title:     "Fifth Post",
			Content:   "Content 5",
			Published: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
			Link:      "https://example.com/5",
		},
		{
			Title:     "Fourth Post",
			Content:   "Content 4",
			Published: time.Date(2025, 3, 13, 5, 0, 15, 0, time.UTC),
			Link:      "https://example.com/4",
		},
		{
			Title:     "Third Post",
			Content:   "Content 3",
			Published: time.Date(2025, 3, 13, 5, 0, 10, 0, time.UTC),
			Link:      "https://example.com/3",
		},
		{
			Title:     "Second Post",
			Content:   "Content 2",
			Published: time.Date(2024, 10, 8, 22, 2, 0, 0, time.UTC),
			Link:      "https://example.com/2",
		},
		{
			Title:     "First Post",
			Content:   "Content 1",
			Published: time.Date(2024, 10, 8, 22, 0, 0, 0, time.UTC),
			Link:      "https://example.com/1",
		},
	}

	normalizeSlice := func(s []string) []string {
		if s == nil {
			return []string{}
		}

		slices.Sort(s)
		return s
	}

	var err error
	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(context.Background(), post)
		if err != nil {
			t.Fatalf("unexpected error while adding posts: %v", err)
		}
	}

	tests := []struct {
		name         string
		currentPage  int
		limit        int
		wantTitles   []string
		wantNumPages int
	}{
		{
			name:         "First page, 3 per page",
			currentPage:  1,
			limit:        3,
			wantTitles:   []string{"Seventh Post", "Sixth Post", "Fifth Post"},
			wantNumPages: 3,
		},
		{
			name:         "Second page, 3 per page",
			currentPage:  2,
			limit:        3,
			wantTitles:   []string{"Fourth Post", "Third Post", "Second Post"},
			wantNumPages: 3,
		},
		{
			name:         "Third page, 3 per page (last page, fewer items)",
			currentPage:  3,
			limit:        3,
			wantTitles:   []string{"First Post"},
			wantNumPages: 3,
		},
		{
			name:         "Page out of range (too high)",
			currentPage:  4,
			limit:        3,
			wantTitles:   []string{},
			wantNumPages: 3,
		},
		{
			name:         "All posts on one page",
			currentPage:  1,
			limit:        10,
			wantTitles:   []string{"Seventh Post", "Sixth Post", "Fifth Post", "Fourth Post", "Third Post", "Second Post", "First Post"},
			wantNumPages: 1,
		},
		{
			name:         "Zero items per page (should handle gracefully)",
			currentPage:  1,
			limit:        0,
			wantTitles:   []string{},
			wantNumPages: 0,
		},
		{
			name:         "Negative page number (should treat as first page)",
			currentPage:  -1,
			limit:        3,
			wantTitles:   []string{"Seventh Post", "Sixth Post", "Fifth Post"},
			wantNumPages: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, pagesNum, err := db.LatestPosts(context.Background(), tt.currentPage, tt.limit)
			if err != nil {
				t.Fatalf("LatestPosts returned error: %v", err)
			}
			if pagesNum != tt.wantNumPages {
				t.Errorf("want pagesNum %d, got %d", tt.wantNumPages, pagesNum)
			}
			var gotTitles []string
			for _, p := range posts {
				gotTitles = append(gotTitles, p.Title)
			}
			gotTitles = normalizeSlice(gotTitles)
			tt.wantTitles = normalizeSlice(tt.wantTitles)
			if !reflect.DeepEqual(gotTitles, tt.wantTitles) {
				t.Errorf("want titles %v, got %v", tt.wantTitles, gotTitles)
			}
		})
	}
}
