package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoldmarkPreview(t *testing.T) {
	s := &server{markdown: newMarkdownRenderer()}
	req := httptest.NewRequest("POST", "/api/preview", strings.NewReader(`{"markdown":"| A | B |\n|---|---|\n| 1 | 2 |"}`))
	recorder := httptest.NewRecorder()
	s.preview(recorder, req)
	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		HTML string `json:"html"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<table>") {
		t.Fatalf("Goldmark table was not rendered: %s", result.HTML)
	}
}
