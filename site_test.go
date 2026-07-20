package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
