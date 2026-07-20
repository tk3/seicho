package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (s *server) posts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	root := s.getRoot()
	if root == "" {
		apiError(w, http.StatusConflict, codedError("site_not_selected", nil))
		return
	}
	items := []postSummary{}
	err := filepath.WalkDir(filepath.Join(root, "content"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".md" && extension != ".markdown" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		front, _, _ := splitDocument(string(data))
		info, _ := entry.Info()
		relative, _ := filepath.Rel(filepath.Join(root, "content"), path)
		items = append(items, postSummary{
			Path:     filepath.ToSlash(relative),
			Title:    field(front, "title", strings.TrimSuffix(entry.Name(), extension)),
			Date:     field(front, "date", ""),
			Draft:    strings.EqualFold(field(front, "draft", "false"), "true"),
			Modified: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("list_posts_failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, items)
}

func (s *server) post(w http.ResponseWriter, r *http.Request) {
	root := s.getRoot()
	if root == "" {
		apiError(w, http.StatusConflict, codedError("site_not_selected", nil))
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getPost(w, r, root)
	case http.MethodPost:
		s.createPost(w, r, root)
	case http.MethodPut:
		s.updatePost(w, r, root)
	case http.MethodDelete:
		s.deletePost(w, r, root)
	default:
		methodNotAllowed(w)
	}
}

func (s *server) getPost(w http.ResponseWriter, r *http.Request, root string) {
	path, err := safePostPath(root, r.URL.Query().Get("path"))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	document, err := readPostDocument(root, path, "read_post_failed")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		apiError(w, status, err)
		return
	}
	jsonResponse(w, http.StatusOK, document)
}

func (s *server) createPost(w http.ResponseWriter, r *http.Request, root string) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	path, err := safePostPath(root, req.Path)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := os.Stat(path); err == nil {
		apiError(w, http.StatusConflict, codedError("post_already_exists", nil))
		return
	}
	relative, _ := filepath.Rel(filepath.Join(root, "content"), path)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "hugo", "new", "content", filepath.ToSlash(relative))
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			apiError(w, http.StatusGatewayTimeout, codedError("hugo_new_timeout", ctx.Err()))
			return
		}
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		apiError(w, http.StatusInternalServerError, codedError("hugo_new_failed", errors.New(message)))
		return
	}
	document, err := readPostDocument(root, path, "read_generated_post_failed")
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusCreated, document)
}

func (s *server) updatePost(w http.ResponseWriter, r *http.Request, root string) {
	var document postDocument
	if err := decodeJSON(r, &document); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	path, err := safePostPath(root, document.Path)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	originalPath := path
	if document.OriginalPath != "" {
		originalPath, err = safePostPath(root, document.OriginalPath)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	renaming := filepath.Clean(originalPath) != filepath.Clean(path)
	if info, err := os.Stat(originalPath); err == nil && document.Modified != "" && fileVersion(info) != document.Modified {
		apiError(w, http.StatusConflict, codedError("file_modified", nil))
		return
	}
	if renaming {
		if _, err := os.Stat(path); err == nil {
			apiError(w, http.StatusConflict, codedError("destination_exists", nil))
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			apiError(w, http.StatusInternalServerError, codedError("inspect_destination_failed", err))
			return
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		apiError(w, http.StatusInternalServerError, codedError("create_post_directory_failed", err))
		return
	}
	delimiter := document.Delimiter
	if delimiter != "+++" {
		delimiter = "---"
	}
	content := delimiter + "\n" + strings.TrimSpace(document.FrontMatter) + "\n" + delimiter + "\n\n" + document.Body
	temporaryPath := path + ".seicho-tmp"
	if err := os.WriteFile(temporaryPath, []byte(content), 0644); err != nil {
		apiError(w, http.StatusInternalServerError, codedError("write_post_failed", err))
		return
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Remove(temporaryPath)
		apiError(w, http.StatusInternalServerError, codedError("replace_post_failed", err))
		return
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("verify_saved_post_failed", err))
		return
	}
	if !bytes.Equal(saved, []byte(content)) {
		apiError(w, http.StatusInternalServerError, codedError("saved_content_mismatch", nil))
		return
	}
	if renaming {
		if err := os.Remove(originalPath); err != nil {
			_ = os.Remove(path)
			apiError(w, http.StatusInternalServerError, codedError("remove_original_post_failed", err))
			return
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("read_file_info_failed", err))
		return
	}
	document.OriginalPath = ""
	document.Modified = fileVersion(info)
	jsonResponse(w, http.StatusOK, document)
}

func (s *server) deletePost(w http.ResponseWriter, r *http.Request, root string) {
	path, err := safePostPath(root, r.URL.Query().Get("path"))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := os.Remove(path); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		apiError(w, status, codedError("delete_post_failed", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
