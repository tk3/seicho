package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerHandlerRoutesAPIWithCommonMiddleware(t *testing.T) {
	handler := (&server{}).handler()
	request := httptest.NewRequest(http.MethodGet, "/api/site", nil)
	request.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"configured":false`) {
		t.Fatalf("site route was not registered: %s", recorder.Body.String())
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security middleware was not applied")
	}
}
