package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type gitChange struct {
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath,omitempty"`
	Status       string `json:"status"`
	Staged       bool   `json:"staged"`
	Unstaged     bool   `json:"unstaged"`
}

type gitStatusResponse struct {
	Repository bool        `json:"repository"`
	Branch     string      `json:"branch,omitempty"`
	Changes    []gitChange `json:"changes"`
}

func (s *server) gitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	root := s.getRoot()
	if root == "" {
		apiError(w, http.StatusConflict, codedError("site_not_selected", nil))
		return
	}
	repository, err := gitRepository(root)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if !repository {
		jsonResponse(w, http.StatusOK, gitStatusResponse{Repository: false, Changes: []gitChange{}})
		return
	}
	branch, err := runGit(r.Context(), root, 15*time.Second, "branch", "--show-current")
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_status_failed", err))
		return
	}
	branchName := strings.TrimSpace(string(branch))
	if branchName == "" {
		commit, commitErr := runGit(r.Context(), root, 15*time.Second, "rev-parse", "--short", "HEAD")
		if commitErr == nil {
			branchName = "detached@" + strings.TrimSpace(string(commit))
		}
	}
	output, err := runGit(r.Context(), root, 15*time.Second, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_status_failed", err))
		return
	}
	changes, err := parseGitStatus(output)
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_status_failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, gitStatusResponse{Repository: true, Branch: branchName, Changes: changes})
}

func (s *server) gitDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	root, ok := s.requireGitRepository(w, r)
	if !ok {
		return
	}
	path, err := safeRepositoryPath(r.URL.Query().Get("path"))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	args := []string{"diff", "--no-ext-diff", "--no-color"}
	if r.URL.Query().Get("staged") == "true" {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	output, err := runGit(r.Context(), root, 15*time.Second, args...)
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_diff_failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"diff": string(output)})
}

func (s *server) gitStage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	root, ok := s.requireGitRepository(w, r)
	if !ok {
		return
	}
	var request struct {
		Path  string `json:"path"`
		Stage bool   `json:"stage"`
	}
	if err := decodeJSON(r, &request); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	path, err := safeRepositoryPath(request.Path)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if request.Stage {
		if _, err := runGit(r.Context(), root, 30*time.Second, "add", "--", path); err != nil {
			apiError(w, http.StatusInternalServerError, codedError("git_stage_failed", err))
			return
		}
	} else if _, err := runGit(r.Context(), root, 30*time.Second, "restore", "--staged", "--", path); err != nil {
		if _, fallbackErr := runGit(r.Context(), root, 30*time.Second, "rm", "--cached", "--", path); fallbackErr != nil {
			apiError(w, http.StatusInternalServerError, codedError("git_unstage_failed", err))
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) gitCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	root, ok := s.requireGitRepository(w, r)
	if !ok {
		return
	}
	var request struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(r, &request); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	message := strings.TrimSpace(request.Message)
	if message == "" {
		apiError(w, http.StatusBadRequest, codedError("git_commit_message_required", nil))
		return
	}
	statusOutput, err := runGit(r.Context(), root, 15*time.Second, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_status_failed", err))
		return
	}
	changes, err := parseGitStatus(statusOutput)
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_status_failed", err))
		return
	}
	hasStaged := false
	for _, change := range changes {
		if change.Staged {
			hasStaged = true
			break
		}
	}
	if !hasStaged {
		apiError(w, http.StatusConflict, codedError("git_nothing_staged", nil))
		return
	}
	output, err := runGit(r.Context(), root, 60*time.Second, "commit", "-m", message)
	if err != nil {
		apiError(w, http.StatusInternalServerError, codedError("git_commit_failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"output": strings.TrimSpace(string(output))})
}

func (s *server) requireGitRepository(w http.ResponseWriter, r *http.Request) (string, bool) {
	root := s.getRoot()
	if root == "" {
		apiError(w, http.StatusConflict, codedError("site_not_selected", nil))
		return "", false
	}
	repository, err := gitRepository(root)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return "", false
	}
	if !repository {
		apiError(w, http.StatusConflict, codedError("git_repository_not_found", nil))
		return "", false
	}
	return root, true
}

func gitRepository(root string) (bool, error) {
	output, err := runGit(context.Background(), root, 15*time.Second, "rev-parse", "--show-toplevel")
	if err != nil {
		var executableError *exec.Error
		if errors.As(err, &executableError) {
			return false, codedError("git_not_installed", err)
		}
		return false, nil
	}
	top := filepath.Clean(filepath.FromSlash(strings.TrimSpace(string(output))))
	selected, err := filepath.Abs(root)
	if err != nil {
		return false, codedError("git_status_failed", err)
	}
	return strings.EqualFold(top, filepath.Clean(selected)), nil
}

func runGit(parent context.Context, root string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err == nil {
		return output, nil
	}
	message := strings.TrimSpace(string(output))
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("git command timed out: %w", ctx.Err())
	}
	if message == "" {
		message = err.Error()
	}
	return nil, fmt.Errorf("%s: %w", message, err)
}

func safeRepositoryPath(path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".git" || strings.HasPrefix(clean, ".git"+string(filepath.Separator)) {
		return "", codedError("git_invalid_path", nil)
	}
	return filepath.ToSlash(clean), nil
}

func parseGitStatus(output []byte) ([]gitChange, error) {
	records := strings.Split(string(output), "\x00")
	changes := make([]gitChange, 0, len(records))
	for index := 0; index < len(records); index++ {
		record := records[index]
		if record == "" {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("invalid git status record %q", record)
		}
		indexStatus, worktreeStatus := record[0], record[1]
		change := gitChange{
			Path:     filepath.ToSlash(record[3:]),
			Status:   record[:2],
			Staged:   indexStatus != ' ' && indexStatus != '?',
			Unstaged: worktreeStatus != ' ' || indexStatus == '?',
		}
		if strings.ContainsRune("RC", rune(indexStatus)) || strings.ContainsRune("RC", rune(worktreeStatus)) {
			index++
			if index >= len(records) || records[index] == "" {
				return nil, fmt.Errorf("missing original path for %q", change.Path)
			}
			change.OriginalPath = filepath.ToSlash(records[index])
		}
		changes = append(changes, change)
	}
	return changes, nil
}
