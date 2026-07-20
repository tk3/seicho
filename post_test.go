package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
