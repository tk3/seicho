package main

import (
	"net/http"
	"os"
	"path/filepath"
)

func (s *server) setRoot(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return codedError("invalid_site_path", err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return codedError("site_folder_not_found", err)
	}
	if !isHugoSite(abs) {
		return codedError("hugo_config_not_found", nil)
	}
	if err := os.MkdirAll(filepath.Join(abs, "content"), 0755); err != nil {
		return codedError("create_content_directory_failed", err)
	}
	s.mu.Lock()
	s.root = abs
	s.mu.Unlock()
	return nil
}

func isHugoSite(root string) bool {
	for _, name := range []string{"hugo.toml", "hugo.yaml", "hugo.yml", "config.toml", "config.yaml", "config.yml"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func (s *server) getRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.root
}

func (s *server) site(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		jsonResponse(w, http.StatusOK, map[string]any{"path": s.getRoot(), "configured": s.getRoot() != "", "version": version})
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.setRoot(req.Path); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"path": s.getRoot(), "configured": true, "version": version})
}
