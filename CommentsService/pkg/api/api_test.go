package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"comments/pkg/models"
	"comments/pkg/mongo"
)

func TestAPI_createCommentHandler(t *testing.T) {
	db, err := mongo.StorageConnect()
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}

	api := New(db)

	t.Cleanup(func() {
		err := mongo.RestoreDB(db)
		if err != nil {
			t.Logf("WARNING: unable to restore DB state after the test: %v", err)
		}

		api.db.Close()
	})

	targetPostID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	var testComment = models.Comment{
		PostID: targetPostID,
		Author: "John Doe",
		Text:   "This is a test comment",
	}

	b, err := json.Marshal(testComment)
	if err != nil {
		t.Fatalf("unexpected error marshaling comment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/comments", bytes.NewBuffer(b))
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("want status code %v, got status code %v", http.StatusCreated, rr.Code)
	}

	var gotComment models.Comment
	b, err = io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(b, &gotComment)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if gotComment.PostID != testComment.PostID {
		t.Errorf("want comment post id %s, got comment post id %s", testComment.PostID, gotComment.PostID)
	}
	if gotComment.Author != testComment.Author {
		t.Errorf("want comment author %s, got comment author %s", testComment.Author, gotComment.Author)
	}
	if gotComment.Text != testComment.Text {
		t.Errorf("want comment text %s, got comment text %s", testComment.Text, gotComment.Text)
	}
	if gotComment.ID == uuid.Nil {
		t.Errorf("comment id has uuid.Nil value")
	}
	if gotComment.Published.IsZero() {
		t.Errorf("comment published has zero time value")
	}
}
