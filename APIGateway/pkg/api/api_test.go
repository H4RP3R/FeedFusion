package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockCommentsService(t *testing.T, statusCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement.
	}))
}

func TestAPI_createCommentProxy(t *testing.T) {
	// TODO: implement.
}
