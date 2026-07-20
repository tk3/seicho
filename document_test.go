package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
