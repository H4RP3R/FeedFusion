package rss

import (
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
)

func TestParser_Run(t *testing.T) {
	testConf := config{
		RSS: []string{"https://3dnews.ru/news/rss/"},
	}

	msgChan := make(chan ParserMsg)
	parser := NewParser(testConf)
	parser.Run(msgChan)

	if msg, ok := <-msgChan; ok {
		if msg.Err != nil {
			t.Errorf("got unexpected error from parser")
		}
		if msg.Source != testConf.RSS[0] {
			t.Errorf("want source %s, got source %s", testConf.RSS[0], msg.Source)
		}
		if len(msg.Data) == 0 {
			t.Errorf("want posts > 0, got %d posts", len(msg.Data))
		}
	}
}

func TestParser_RunBrokenSource(t *testing.T) {
	testConf := config{
		RSS: []string{"https://example.xyz/invalid/rss/"},
	}

	msgChan := make(chan ParserMsg)
	parser := NewParser(testConf)
	parser.Run(msgChan)

	if msg, ok := <-msgChan; ok {
		if msg.Err == nil {
			t.Errorf("expected error, got %v", msg.Err)
		}
		if msg.Source != testConf.RSS[0] {
			t.Errorf("want source %s, got source %s", testConf.RSS[0], msg.Source)
		}
		if len(msg.Data) != 0 {
			t.Errorf("expected empty data, got %d posts", len(msg.Data))
		}
	}
}

func TestHandleFeed(t *testing.T) {
	mockFeed := &gofeed.Feed{
		Items: []*gofeed.Item{
			{
				Title:       "Test Post 1",
				Description: "Test Content 1",
				Link:        "https://example.com/1",
				Published:   "Wed, 01 May 2024 12:00:00 GMT", // RFC1123 format
			},
			{
				Title:       "Test Post 2",
				Description: "", // Empty content test case
				Link:        "https://example.com/2",
				Published:   "2024-05-01T15:00:00Z", // RFC3339 format
			},
			{
				Title:       "", // Empty title test case
				Description: "Content without title",
				Link:        "",
				Published:   "invalid date format", // Bad date format
			},
		},
	}

	parser := &Parser{}
	posts, err := parser.handleFeed(mockFeed)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test post count
	if len(posts) != 3 {
		t.Fatalf("Expected 3 posts, got %d", len(posts))
	}

	// Test first post conversion
	t.Run("ValidPostConversion", func(t *testing.T) {
		post := posts[0]
		wantPubTime, _ := time.Parse(time.RFC1123, "Wed, 01 May 2024 12:00:00 GMT")

		wantPostTitle := mockFeed.Items[0].Title
		if post.Title != wantPostTitle {
			t.Errorf("want post title '%s', got post title '%s'", wantPostTitle, post.Title)
		}
		wantPostContent := mockFeed.Items[0].Description
		if post.Content != wantPostContent {
			t.Errorf("want post content '%s', got post content '%s'", wantPostContent, post.Content)
		}
		if !post.Published.Equal(wantPubTime.UTC()) {
			t.Errorf("want post published %v, got post published %v", wantPubTime.UTC(), post.Published)
		}
	})

	// Test empty content handling
	t.Run("EmptyContentHandling", func(t *testing.T) {
		if posts[1].Content != "" {
			t.Errorf("Expected empty content, got %q", posts[1].Content)
		}
	})

	// Test invalid date handling
	t.Run("InvalidDateHandling", func(t *testing.T) {
		if !posts[2].Published.IsZero() {
			t.Error("Expected zero time for invalid date format")
		}
	})
}
