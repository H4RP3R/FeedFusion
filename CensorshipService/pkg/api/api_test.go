package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	"censorship/pkg/censor"
	"censorship/pkg/models"
)

const testRequestID = "9b4f6c5d-1a32-4d8f-b5a6-23c9e1f7d2a1"

func TestMain(m *testing.M) {
	log.SetLevel(log.PanicLevel)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestAPI_checkComment(t *testing.T) {
	censor := &censor.Censor{}
	err := censor.LoadFromJSON("../censor/test_data/words.json")
	if err != nil {
		t.Fatalf("failed to load words for censor: %v", err)
	}

	api, err := New("", censor, nil)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

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

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(b))
	req.Header.Set("X-Request-Id", testRequestID)
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want status code %v, got status code %v", http.StatusOK, rr.Code)
	}
}

func TestAPI_checkCommentBanned(t *testing.T) {
	censor := &censor.Censor{}
	err := censor.LoadFromJSON("../censor/test_data/words.json")
	if err != nil {
		t.Fatalf("failed to load words for censor: %v", err)
	}

	api, err := New("", censor, nil)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	targetPostID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	var testComment = models.Comment{
		PostID: targetPostID,
		Author: "John Doe",
		Text:   "Нескрепный коммент",
	}

	b, err := json.Marshal(testComment)
	if err != nil {
		t.Fatalf("failed to marshal comment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(b))
	req.Header.Set("X-Request-Id", testRequestID)
	rr := httptest.NewRecorder()
	api.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want status code %v, got status code %v", http.StatusUnprocessableEntity, rr.Code)
	}
}
