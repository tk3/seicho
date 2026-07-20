package main

import (
	"embed"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed web/*
var webFiles embed.FS

func static(w http.ResponseWriter, r *http.Request) {
	name := "web" + r.URL.Path
	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/edit/") {
		name = "web/index.html"
	}
	b, err := webFiles.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(name)))
	_, _ = w.Write(b)
}
