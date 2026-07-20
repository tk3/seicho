package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseGitStatus(t *testing.T) {
	output := []byte(" M content/modified.md\x00A  content/staged.md\x00?? content/new.md\x00R  content/renamed.md\x00content/original.md\x00")
	changes, err := parseGitStatus(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 4 {
		t.Fatalf("changes = %#v", changes)
	}
	if changes[0].Path != "content/modified.md" || changes[0].Staged || !changes[0].Unstaged {
		t.Fatalf("unexpected modified change: %#v", changes[0])
	}
	if changes[1].Path != "content/staged.md" || !changes[1].Staged || changes[1].Unstaged {
		t.Fatalf("unexpected staged change: %#v", changes[1])
	}
	if changes[2].Path != "content/new.md" || changes[2].Staged || !changes[2].Unstaged {
		t.Fatalf("unexpected untracked change: %#v", changes[2])
	}
	if changes[3].Path != "content/renamed.md" || changes[3].OriginalPath != "content/original.md" || !changes[3].Staged {
		t.Fatalf("unexpected renamed change: %#v", changes[3])
	}
}

func TestParseGitStatusRejectsMalformedRecords(t *testing.T) {
	if _, err := parseGitStatus([]byte("invalid\x00")); err == nil {
		t.Fatal("malformed status was accepted")
	}
	if _, err := parseGitStatus([]byte("R  renamed.md\x00")); err == nil {
		t.Fatal("rename without original path was accepted")
	}
}

func TestSafeRepositoryPath(t *testing.T) {
	for _, valid := range []string{"content/post.md", "hugo.toml", "themes/example/layout.html"} {
		if _, err := safeRepositoryPath(valid); err != nil {
			t.Errorf("safeRepositoryPath rejected %q: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", ".", "..", "../secret", ".git/config", "C:/Windows/file"} {
		if _, err := safeRepositoryPath(invalid); err == nil {
			t.Errorf("safeRepositoryPath accepted %q", invalid)
		}
	}
}

func TestGitStatusDiffStageAndCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	postPath := filepath.Join(root, "content", "post.md")
	if err := os.MkdirAll(filepath.Dir(postPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(postPath, []byte("Original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init"}, {"config", "user.name", "Seicho Test"}, {"config", "user.email", "seicho@example.invalid"}, {"add", "--", "content/post.md"}, {"commit", "-m", "Initial"}} {
		if _, err := runGit(context.Background(), root, 30*time.Second, args...); err != nil {
			t.Fatalf("git %s: %v", strings.Join(args, " "), err)
		}
	}
	if err := os.WriteFile(postPath, []byte("Updated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	s := &server{root: root}

	statusRecorder := httptest.NewRecorder()
	s.gitStatus(statusRecorder, httptest.NewRequest(http.MethodGet, "/api/git/status", nil))
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status API = %d: %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var status gitStatusResponse
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Repository || len(status.Changes) != 1 || !status.Changes[0].Unstaged {
		t.Fatalf("unexpected status: %#v", status)
	}

	diffRecorder := httptest.NewRecorder()
	s.gitDiff(diffRecorder, httptest.NewRequest(http.MethodGet, "/api/git/diff?path=content%2Fpost.md", nil))
	if diffRecorder.Code != http.StatusOK || !strings.Contains(diffRecorder.Body.String(), "+Updated") {
		t.Fatalf("unexpected diff: status=%d body=%s", diffRecorder.Code, diffRecorder.Body.String())
	}

	unstagedCommitRecorder := httptest.NewRecorder()
	s.gitCommit(unstagedCommitRecorder, httptest.NewRequest(http.MethodPost, "/api/git/commit", strings.NewReader(`{"message":"Update post"}`)))
	if unstagedCommitRecorder.Code != http.StatusConflict || !strings.Contains(unstagedCommitRecorder.Body.String(), "git_nothing_staged") {
		t.Fatalf("unstaged commit API = %d: %s", unstagedCommitRecorder.Code, unstagedCommitRecorder.Body.String())
	}

	stageRecorder := httptest.NewRecorder()
	s.gitStage(stageRecorder, httptest.NewRequest(http.MethodPost, "/api/git/stage", strings.NewReader(`{"path":"content/post.md","stage":true}`)))
	if stageRecorder.Code != http.StatusNoContent {
		t.Fatalf("stage API = %d: %s", stageRecorder.Code, stageRecorder.Body.String())
	}
	unstageRecorder := httptest.NewRecorder()
	s.gitStage(unstageRecorder, httptest.NewRequest(http.MethodPost, "/api/git/stage", strings.NewReader(`{"path":"content/post.md","stage":false}`)))
	if unstageRecorder.Code != http.StatusNoContent {
		t.Fatalf("unstage API = %d: %s", unstageRecorder.Code, unstageRecorder.Body.String())
	}
	stageRecorder = httptest.NewRecorder()
	s.gitStage(stageRecorder, httptest.NewRequest(http.MethodPost, "/api/git/stage", strings.NewReader(`{"path":"content/post.md","stage":true}`)))
	if stageRecorder.Code != http.StatusNoContent {
		t.Fatalf("second stage API = %d: %s", stageRecorder.Code, stageRecorder.Body.String())
	}

	emptyCommitRecorder := httptest.NewRecorder()
	s.gitCommit(emptyCommitRecorder, httptest.NewRequest(http.MethodPost, "/api/git/commit", strings.NewReader(`{"message":" "}`)))
	if emptyCommitRecorder.Code != http.StatusBadRequest {
		t.Fatalf("empty commit API = %d: %s", emptyCommitRecorder.Code, emptyCommitRecorder.Body.String())
	}

	commitRecorder := httptest.NewRecorder()
	s.gitCommit(commitRecorder, httptest.NewRequest(http.MethodPost, "/api/git/commit", strings.NewReader(`{"message":"Update post"}`)))
	if commitRecorder.Code != http.StatusOK {
		t.Fatalf("commit API = %d: %s", commitRecorder.Code, commitRecorder.Body.String())
	}

	finalStatusRecorder := httptest.NewRecorder()
	s.gitStatus(finalStatusRecorder, httptest.NewRequest(http.MethodGet, "/api/git/status", nil))
	if err := json.Unmarshal(finalStatusRecorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status.Changes) != 0 {
		t.Fatalf("changes remain after commit: %#v", status.Changes)
	}
}
