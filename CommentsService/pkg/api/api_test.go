package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"comments/pkg/models"
	"comments/pkg/mongo"
)

const testRequestID = "9b4f6c5d-1a32-4d8f-b5a6-23c9e1f7d2a1"

func TestAPI_createCommentHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := mongo.StorageConnect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}

	api := New(db)

	t.Cleanup(func() {
		err := mongo.RestoreDB(db)
		if err != nil {
			t.Logf("WARNING: unable to restore DB state after the test: %v", err)
		}

		db.Close(ctx)
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
		t.Fatalf("failed to marshal comment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/comments", bytes.NewBuffer(b))
	req.Header.Set("X-Request-Id", testRequestID)
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

func TestAPI_commentsHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := mongo.StorageConnect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}

	api := New(db)

	t.Cleanup(func() {
		err := mongo.RestoreDB(db)
		if err != nil {
			t.Logf("WARNING: unable to restore DB state after the test: %v", err)
		}

		db.Close(ctx)
	})

	targetPostID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}

	testComment := models.Comment{
		PostID: targetPostID,
		Author: "John Doe",
		Text:   "This is a test comment",
	}

	b, err := json.Marshal(testComment)
	if err != nil {
		t.Fatalf("failed to marshal comment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/comments", bytes.NewBuffer(b))
	req.Header.Set("X-Request-Id", testRequestID)
	rr := httptest.NewRecorder()
	api.r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want status code %v, got status code %v", http.StatusCreated, rr.Code)
	}
	var targetComment models.Comment
	b, err = io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	err = json.Unmarshal(b, &targetComment)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	reqUrl := fmt.Sprintf("/comments?post_id=%s", targetPostID.String())
	req = httptest.NewRequest(http.MethodGet, reqUrl, nil)
	req.Header.Set("X-Request-Id", testRequestID)
	rr = httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want status code %v, got status code %v", http.StatusOK, rr.Code)
	}
	var comments []models.Comment
	b, err = io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	err = json.Unmarshal(b, &comments)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(comments) == 0 {
		t.Fatalf("expected at least one comment, got none")
	}
	gotComment := comments[0]

	// Normalize times before comparison
	targetComment.Published = targetComment.Published.Round(time.Second).UTC()
	gotComment.Published = gotComment.Published.Round(time.Second).UTC()

	if !reflect.DeepEqual(gotComment, targetComment) {
		t.Errorf("want comment\n%+v\n\ngot comment\n%+v\n", targetComment, gotComment)
	}
}

func TestAPI_commentsHandlerNoComments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := mongo.StorageConnect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	defer db.Close(ctx)

	api := New(db)

	targetPostID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	reqUrl := fmt.Sprintf("/comments?post_id=%s", targetPostID.String())
	req := httptest.NewRequest(http.MethodGet, reqUrl, nil)
	req.Header.Set("X-Request-Id", testRequestID)
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want status code %v, got status code %v", http.StatusNotFound, rr.Code)
	}
}
