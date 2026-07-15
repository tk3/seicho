package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestSplitDocument(t *testing.T) {
	tests := []struct{ source, front, body, delim string }{
		{"---\ntitle: Test\n---\nHello", "title: Test", "Hello", "---"},
		{"+++\ntitle = 'Test'\n+++\nHello", "title = 'Test'", "Hello", "+++"},
		{"Hello", "", "Hello", ""},
	}
	for _, tt := range tests {
		front, body, delim := splitDocument(tt.source)
		if front != tt.front || body != tt.body || delim != tt.delim {
			t.Fatalf("splitDocument(%q) = %q, %q, %q", tt.source, front, body, delim)
		}
	}
}

func TestSafePostPath(t *testing.T) {
	root := t.TempDir()
	if _, err := safePostPath(root, "posts/hello.md"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../secret.md", "post.txt", "C:/Windows/file.md"} {
		if _, err := safePostPath(root, path); err == nil {
			t.Errorf("safePostPath accepted %q", path)
		}
	}
}

func TestSetRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hugo.toml"), []byte("baseURL='/'"), 0644); err != nil {
		t.Fatal(err)
	}
	s := &server{}
	if err := s.setRoot(root); err != nil {
		t.Fatal(err)
	}
	if s.getRoot() != root {
		t.Fatalf("root = %q", s.getRoot())
	}
	if _, err := os.Stat(filepath.Join(root, "content")); err != nil {
		t.Fatal("content directory was not created")
	}
}
