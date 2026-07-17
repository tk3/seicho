package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestAccessTraceWritesRelativeURLAndStatus(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/posts?q=draft", nil))
	if got := output.String(); !strings.Contains(got, "] GET /api/posts?q=draft 201 ") {
		t.Fatalf("unexpected trace: %q", got)
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID was not set")
	}
}

func TestAccessTraceDefaultsStatusToOK(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got := output.String(); !strings.Contains(got, "] GET / 200 ") {
		t.Fatalf("unexpected trace: %q", got)
	}
}

func TestAccessTraceIncludesAPIError(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusInternalServerError, errors.New("disk write failed"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PUT", "/api/post", nil))
	if got := output.String(); !strings.Contains(got, `] PUT /api/post 500 `) || !strings.Contains(got, `error="disk write failed"`) {
		t.Fatalf("unexpected error trace: %q", got)
	}
}

func TestAPIErrorUsesRequestedLanguage(t *testing.T) {
	handler := languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusBadRequest, errors.New("投稿パスが不正です"))
	}))
	req := httptest.NewRequest("GET", "/api/post", nil)
	req.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if got := recorder.Body.String(); !strings.Contains(got, "The post path is invalid.") {
		t.Fatalf("error was not translated: %s", got)
	}
}

func TestAPIErrorDefaultsToJapanese(t *testing.T) {
	handler := languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusBadRequest, errors.New("投稿パスが不正です"))
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/post", nil))
	if got := recorder.Body.String(); !strings.Contains(got, "投稿パスが不正です") {
		t.Fatalf("unexpected default language: %s", got)
	}
}

func TestAccessTraceRecoversPanic(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := output.String(); !strings.Contains(got, "PANIC boom") || !strings.Contains(got, "TestAccessTraceRecoversPanic") || !strings.Contains(got, `error="panic: boom"`) {
		t.Fatalf("panic trace is incomplete: %q", got)
	}
}

func TestStartupTraceIncludesRuntimeInformation(t *testing.T) {
	var output bytes.Buffer
	writeStartupTrace(&output, "127.0.0.1:1221", "/path/to/site")
	for _, expected := range []string{
		"Seicho " + version,
		"OS: " + runtime.GOOS + "/" + runtime.GOARCH,
		"Go: " + runtime.Version(),
		"Listen: http://127.0.0.1:1221",
		"Site: /path/to/site",
		"Trace: enabled",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("startup trace does not contain %q: %s", expected, output.String())
		}
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

func TestChangingPostPathRenamesFile(t *testing.T) {
	root := t.TempDir()
	contentDir := filepath.Join(root, "content", "posts")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(contentDir, "old.md")
	newPath := filepath.Join(contentDir, "renamed.md")
	if err := os.WriteFile(oldPath, []byte("---\ntitle: Test\n---\n\nBody"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := postDocument{Path: "posts/renamed.md", OriginalPath: "posts/old.md", FrontMatter: "title: Renamed", Body: "Updated", Delimiter: "---", Modified: fileVersion(info)}
	payload, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := &server{root: root, markdown: newMarkdownRenderer()}
	recorder := httptest.NewRecorder()
	s.post(recorder, httptest.NewRequest("PUT", "/api/post", strings.NewReader(string(payload))))
	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old file still exists: %v", err)
	}
	saved, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "Updated") {
		t.Fatalf("renamed content was not saved: %s", saved)
	}
}
