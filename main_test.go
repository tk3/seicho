package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
