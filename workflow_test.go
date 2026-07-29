package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPostLifecycleThroughAPI(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hugo.toml"), []byte("baseURL = '/'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	postPath := filepath.Join(root, "content", "posts", "existing.md")
	if err := os.MkdirAll(filepath.Dir(postPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(postPath, []byte("---\ntitle: Existing\ndate: 2026-07-29\ndraft: true\n---\n\nOriginal"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &server{markdown: newMarkdownRenderer()}
	handler := s.handler()

	configure := performJSONRequest(t, handler, http.MethodPost, "/api/site", map[string]string{"path": root})
	if configure.Code != http.StatusOK {
		t.Fatalf("configure site: status=%d body=%s", configure.Code, configure.Body.String())
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/posts", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list posts: status=%d body=%s", list.Code, list.Body.String())
	}
	var summaries []postSummary
	if err := json.Unmarshal(list.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Path != "posts/existing.md" || summaries[0].Title != "Existing" || !summaries[0].Draft {
		t.Fatalf("unexpected post summaries: %#v", summaries)
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/post?path=posts%2Fexisting.md", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("get post: status=%d body=%s", get.Code, get.Body.String())
	}
	var document postDocument
	if err := json.Unmarshal(get.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	document.OriginalPath = document.Path
	document.Path = "posts/renamed.md"
	document.Body = "Updated"

	update := performJSONRequest(t, handler, http.MethodPut, "/api/post", document)
	if update.Code != http.StatusOK {
		t.Fatalf("update post: status=%d body=%s", update.Code, update.Body.String())
	}
	if _, err := os.Stat(postPath); !os.IsNotExist(err) {
		t.Fatalf("original post remains after rename: %v", err)
	}
	renamedPath := filepath.Join(root, "content", "posts", "renamed.md")
	if saved, err := os.ReadFile(renamedPath); err != nil || !bytes.Contains(saved, []byte("Updated")) {
		t.Fatalf("renamed post was not saved: content=%q err=%v", saved, err)
	}

	remove := httptest.NewRecorder()
	handler.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/post?path=posts%2Frenamed.md", nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete post: status=%d body=%s", remove.Code, remove.Body.String())
	}
	if _, err := os.Stat(renamedPath); !os.IsNotExist(err) {
		t.Fatalf("post remains after delete: %v", err)
	}

	list = httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/posts", nil))
	if err := json.Unmarshal(list.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 0 {
		t.Fatalf("deleted post remains in list: %#v", summaries)
	}
}

func TestCreatePostThroughAPI(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content"), 0755); err != nil {
		t.Fatal(err)
	}
	s := &server{
		root:     root,
		markdown: newMarkdownRenderer(),
		hugoNew: func(_ context.Context, commandRoot, relative string) ([]byte, error) {
			if commandRoot != root || relative != "posts/new.md" {
				t.Fatalf("unexpected Hugo command: root=%q relative=%q", commandRoot, relative)
			}
			path := filepath.Join(commandRoot, "content", filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return nil, err
			}
			return nil, os.WriteFile(path, []byte("---\ntitle: New post\ndraft: true\n---\n\nWrite here"), 0644)
		},
	}

	create := performJSONRequest(t, s.handler(), http.MethodPost, "/api/post", map[string]string{"path": "posts/new.md"})
	if create.Code != http.StatusCreated {
		t.Fatalf("create post: status=%d body=%s", create.Code, create.Body.String())
	}
	var document postDocument
	if err := json.Unmarshal(create.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Path != "posts/new.md" || document.Body != "Write here" || document.FrontMatter == "" {
		t.Fatalf("unexpected created post: %#v", document)
	}
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path string, value any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewReader(payload)))
	return recorder
}
