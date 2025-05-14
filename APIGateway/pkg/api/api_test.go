package api

import (
	"bytes"
	"encoding/json"
	"gateway/pkg/models"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestAPI_filterNewsProxy(t *testing.T) {
	defer gock.Off()

	responseNews := []models.Post{
		{
			ID:        uuid.FromStringOrNil("f3767624-65e9-5e26-80e1-aea970710389"),
			Title:     "The Rise of AI Code Assistants: Revolutionizing Software Development",
			Content:   `The world of software development is undergoing a significant transformation with the emergence of AI code assistants. These intelligent tools are designed to assist developers in writing, debugging, and optimizing their code, making the development process faster, more efficient, and more enjoyable.`,
			Published: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
			Link:      "https://tech/posts/1198",
		},
	}

	api := New()

	gock.New(newsServiceURL).Reply(http.StatusOK).JSON(responseNews)

	req := httptest.NewRequest(http.MethodGet, "/news/filter?contains=AI", nil)
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got status code %v", http.StatusOK, rr.Code)
	}

	var gotNews []models.Post
	b, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	err = json.Unmarshal(b, &gotNews)
	if err != nil {
		t.Errorf("failed to unmarshal response body: %v", err)
	}

	if !reflect.DeepEqual(responseNews, gotNews) {
		t.Errorf("want response news\n%+v\n\ngot response news\n%+v\n", responseNews, gotNews)
	}
}

func TestAPI_newsDetailedProxy(t *testing.T) {
	defer gock.Off()

	targetPostID := uuid.FromStringOrNil("f3767624-65e9-5e26-80e1-aea970710389")
	postSample := models.Post{
		ID:        targetPostID,
		Title:     "The Rise of AI Code Assistants: Revolutionizing Software Development",
		Content:   `The world of software development is undergoing a significant transformation with the emergence of AI code assistants. These intelligent tools are designed to assist developers in writing, debugging, and optimizing their code, making the development process faster, more efficient, and more enjoyable.`,
		Published: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		Link:      "https://tech/posts/1198",
	}
	commentsSample := []models.Comment{
		{
			ID:        uuid.FromStringOrNil("9b4f6c5d-1a32-4d8f-b5a6-23c9e1f7d2a1"),
			PostID:    targetPostID,
			Author:    "John Doe",
			Text:      "This is a test comment",
			Published: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	api := New()

	gock.New(newsServiceURL).Reply(http.StatusOK).JSON(postSample)
	gock.New(commentsServiceURL).Reply(http.StatusOK).JSON(commentsSample)

	req := httptest.NewRequest(http.MethodGet, "/news/"+targetPostID.String(), nil)
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got status code %v", http.StatusOK, rr.Code)
	}

	var gotPost models.Post
	b, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	err = json.Unmarshal(b, &gotPost)
	if err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	postSample.Comments = commentsSample
	if !reflect.DeepEqual(postSample, gotPost) {
		t.Errorf("want response post\n%+v\n\ngot response post\n%+v\n", postSample, gotPost)
	}
}
