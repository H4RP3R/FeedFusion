package api

import (
	"bytes"
	"encoding/json"
	"gateway/pkg/models"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/h2non/gock"
)

func TestAPI_createCommentProxy(t *testing.T) {
	defer gock.Off()

	api := New()

	targetPostID, _ := uuid.NewV4()
	testComment := models.Comment{
		PostID: targetPostID,
		Author: "Some Dude",
		Text:   "Something interesting",
	}
	b, err := json.Marshal(testComment)
	if err != nil {
		t.Errorf("failed to marshal post: %v", err)
	}

	gock.New(commentsServiceURL).
		Reply(http.StatusCreated).
		JSON(map[string]string{
			"id":        uuid.NewV5(uuid.NamespaceURL, testComment.Author+testComment.Text).String(),
			"post_id":   testComment.PostID.String(),
			"author":    testComment.Author,
			"text":      testComment.Text,
			"published": time.Now().UTC().Format(time.RFC3339),
		})

	req := httptest.NewRequest(http.MethodPost, "/comments", bytes.NewBuffer(b))
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want status code %v, got status code %v", http.StatusCreated, rr.Code)
	}

	b, err = io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var gotComment models.Comment
	err = json.Unmarshal(b, &gotComment)
	if err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if gotComment.PostID != testComment.PostID {
		t.Errorf("want post_id %v, got %v", testComment.PostID, gotComment.PostID)
	}
	if gotComment.Author != testComment.Author {
		t.Errorf("want author %q, got %q", testComment.Author, gotComment.Author)
	}
	if gotComment.Text != testComment.Text {
		t.Errorf("want text %q, got %q", testComment.Text, gotComment.Text)
	}
	if gotComment.ID == uuid.Nil {
		t.Error("want non-nil comment ID")
	}
	if gotComment.Published.IsZero() {
		t.Error("want non-zero published time")
	}
}

// TODO: add negative scenario test for createCommentProxy.
