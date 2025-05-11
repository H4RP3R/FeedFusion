package storage

import (
	"os"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gofrs/uuid"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.PanicLevel)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestValidatePosts(t *testing.T) {
	testPosts := []Post{
		{
			// Empty title
			Title:     "",
			Content:   "Sample content",
			Published: time.Now(),
			Link:      "https://example.com/post/1",
		},
		{
			// Empty content
			Title:     "Sample Title",
			Content:   "",
			Published: time.Now(),
			Link:      "https://example.com/post/2",
		},
		{
			// Default time
			Title:     "Sample Title",
			Content:   "Sample content",
			Published: time.Time{},
			Link:      "https://example.com/post/3",
		},
		{
			// Empty link
			Title:     "Sample Title",
			Content:   "Sample content",
			Published: time.Now(),
			Link:      "",
		},
		{
			// Valid post
			Title:     "Sample Title",
			Content:   "Sample content",
			Published: time.Now(),
			Link:      "https://example.com/post/5",
		},
		{
			// Invalid url
			Title:     "Valid Title",
			Content:   "Valid Content",
			Published: time.Now(),
			Link:      "invalid_url",
		},
	}

	for i, post := range testPosts {
		testPosts[i].ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
	}

	gotPosts := ValidatePosts(testPosts...)
	if len(gotPosts) != 1 {
		t.Fatalf("want 1 post after validation, got %d posts after validation", len(gotPosts))
	}

	gotValidPost := gotPosts[0]
	wantValidPost := testPosts[4]
	if !reflect.DeepEqual(gotValidPost, wantValidPost) {
		t.Errorf("want valid post ID:%v, got valid post ID:%v", wantValidPost, gotValidPost)
	}
}
