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
		{"---\ntitle: Test\n---\n\nHello", "title: Test", "Hello", "---"},
		{"---\ntitle: Test\n---\n\n\nHello", "title: Test", "\nHello", "---"},
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

func TestFileVersionSurvivesJSONRoundTrip(t *testing.T) {
	doc := postDocument{Modified: "2026-07-15T09:12:34.123456789Z"}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded postDocument
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Modified != doc.Modified {
		t.Fatalf("modified changed from %q to %q", doc.Modified, decoded.Modified)
	}
}

func TestExistingPostCanBeSavedRepeatedly(t *testing.T) {
	root := t.TempDir()
	contentDir := filepath.Join(root, "content", "posts")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(contentDir, "existing.md")
	initial := "---\ntitle: Existing\n---\n\nOriginal"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	s := &server{root: root, markdown: newMarkdownRenderer()}

	for _, body := range []string{"First update", "Second update"} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		doc := postDocument{Path: "posts/existing.md", FrontMatter: "title: Existing", Body: body, Delimiter: "---", Modified: fileVersion(info)}
		payload, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("PUT", "/api/post", strings.NewReader(string(payload)))
		recorder := httptest.NewRecorder()
		s.post(recorder, req)
		if recorder.Code != 200 {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		saved, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(saved), body) {
			t.Fatalf("saved file does not contain %q: %s", body, saved)
		}
	}
}

func TestRemovingLeadingBodyBlankLineIsSaved(t *testing.T) {
	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(contentDir, "post.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Test\n---\n\n\nBody"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	s := &server{root: root, markdown: newMarkdownRenderer()}
	doc := postDocument{Path: "post.md", FrontMatter: "title: Test", Body: "Body", Delimiter: "---", Modified: fileVersion(info)}
	payload, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	s.post(recorder, httptest.NewRequest("PUT", "/api/post", strings.NewReader(string(payload))))
	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != "---\ntitle: Test\n---\n\nBody" {
		t.Fatalf("unexpected saved content: %q", saved)
	}
}
