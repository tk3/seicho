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

func TestStaticServesEditorRoutes(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest("GET", "/edit/posts/hello.md", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "<title>Seicho</title>") {
		t.Fatal("editor route did not serve the application shell")
	}
	if !strings.Contains(recorder.Body.String(), `<img class="mark" src="/favicon.svg" alt="">`) {
		t.Fatal("application shell does not use the favicon as its header icon")
	}
	if !strings.Contains(recorder.Body.String(), `placeholder="/path/to/hugo-site"`) {
		t.Fatal("site path placeholder is not platform-neutral")
	}
	shell := recorder.Body.String()
	languageIndex := strings.Index(shell, `id="language"`)
	siteIndex := strings.Index(shell, `id="change-site"`)
	toolsIndex := strings.Index(shell, `id="tools-menu"`)
	if languageIndex < 0 || siteIndex < languageIndex || toolsIndex < siteIndex {
		t.Fatal("header controls are not ordered as language, site change, and tools")
	}
	if !strings.Contains(shell, `<summary data-i18n-aria="tools" data-i18n-title="tools">…</summary>`) {
		t.Fatal("tools menu does not use the transparent ellipsis trigger")
	}
	for _, asset := range []string{"/favicon.svg", "/tokens.css", "/interface-theme.css", "/router.js", "/i18n.js", "/editor-view.js", "/git-panel.css", "/git-panel.js"} {
		if !strings.Contains(recorder.Body.String(), asset) {
			t.Errorf("application shell does not reference %s", asset)
		}
	}
}

func TestStaticServesRefactoredAssets(t *testing.T) {
	for _, path := range []string{"/favicon.svg", "/tokens.css", "/interface-theme.css", "/router.js", "/i18n.js", "/editor-view.js", "/git-panel.css", "/git-panel.js"} {
		recorder := httptest.NewRecorder()
		static(recorder, httptest.NewRequest("GET", path, nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d", path, recorder.Code)
		}
	}
}

func TestFaviconUsesApplicationAccentColor(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/favicon.svg", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `fill="#da5b3a"`) {
		t.Fatalf("favicon does not use the application accent color: status=%d body=%s", recorder.Code, recorder.Body.String())
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
		apiError(w, http.StatusInternalServerError, codedError("write_post_failed", errors.New("disk write failed")))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PUT", "/api/post", nil))
	if got := output.String(); !strings.Contains(got, `] PUT /api/post 500 `) || !strings.Contains(got, `error="write_post_failed: disk write failed"`) {
		t.Fatalf("unexpected error trace: %q", got)
	}
}

func TestAPIErrorUsesRequestedLanguage(t *testing.T) {
	handler := languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusBadRequest, codedError("invalid_post_path", nil))
	}))
	req := httptest.NewRequest("GET", "/api/post", nil)
	req.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if got := recorder.Body.String(); !strings.Contains(got, `"code":"invalid_post_path"`) || !strings.Contains(got, "The post path is invalid.") {
		t.Fatalf("error was not translated: %s", got)
	}
}

func TestAPIErrorDefaultsToJapanese(t *testing.T) {
	handler := languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusBadRequest, codedError("invalid_post_path", nil))
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/post", nil))
	if got := recorder.Body.String(); !strings.Contains(got, "投稿パスが不正です") {
		t.Fatalf("unexpected default language: %s", got)
	}
}

func TestLocalizedAPIErrorPreservesTraceMetadata(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusInternalServerError, codedError("write_post_failed", errors.New("disk full")))
	})))
	req := httptest.NewRequest("PUT", "/api/post", nil)
	req.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if got := recorder.Body.String(); !strings.Contains(got, `"code":"write_post_failed"`) || !strings.Contains(got, `"requestId":`) {
		t.Fatalf("response metadata is incomplete: %s", got)
	}
	if got := output.String(); !strings.Contains(got, `error="write_post_failed: disk full"`) {
		t.Fatalf("trace error is incomplete: %s", got)
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

func TestCreatePostRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content"), 0755); err != nil {
		t.Fatal(err)
	}
	s := &server{root: root}
	absPath := filepath.Join(t.TempDir(), "absolute.md")
	tests := []struct {
		name string
		path string
		code string
	}{
		{"parent traversal", "../outside.md", "invalid_post_path"},
		{"nested traversal", "posts/../../outside.md", "invalid_post_path"},
		{"absolute path", absPath, "invalid_post_path"},
		{"invalid extension", "posts/outside.txt", "invalid_post_extension"},
		{"empty path", "", "invalid_post_path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]string{"path": tt.path})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			s.post(recorder, httptest.NewRequest(http.MethodPost, "/api/post", bytes.NewReader(payload)))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"code":"`+tt.code+`"`) {
				t.Fatalf("expected error code %q, body = %s", tt.code, recorder.Body.String())
			}
		})
	}
	if _, err := os.Stat(filepath.Join(root, "..", "outside.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path traversal created a file outside content: %v", err)
	}
	if _, err := os.Stat(absPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("absolute path created a file: %v", err)
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

func TestReadPostDocumentBuildsEditorDocument(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "content", "posts", "hello.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\ntitle: Hello\n---\n\nBody"), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := readPostDocument(root, path, "read_post_failed")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Path != "posts/hello.md" || doc.FrontMatter != "title: Hello" || doc.Body != "Body" || doc.Delimiter != "---" || doc.Modified == "" {
		t.Fatalf("unexpected document: %#v", doc)
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
