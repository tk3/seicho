package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readPostDocument(root, path, readErrorCode string) (postDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return postDocument{}, codedError(readErrorCode, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return postDocument{}, codedError("read_file_info_failed", err)
	}
	front, body, delimiter := splitDocument(string(data))
	relative, _ := filepath.Rel(filepath.Join(root, "content"), path)
	return postDocument{
		Path:        filepath.ToSlash(relative),
		FrontMatter: front,
		Body:        body,
		Delimiter:   delimiter,
		Modified:    fileVersion(info),
	}, nil
}

func fileVersion(info os.FileInfo) string {
	return info.ModTime().UTC().Format(time.RFC3339Nano)
}

func safePostPath(root, relative string) (string, error) {
	relative = filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if relative == "." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || relative == ".." {
		return "", codedError("invalid_post_path", nil)
	}
	extension := strings.ToLower(filepath.Ext(relative))
	if extension != ".md" && extension != ".markdown" {
		return "", codedError("invalid_post_extension", nil)
	}
	base := filepath.Join(root, "content")
	target := filepath.Join(base, relative)
	if back, _ := filepath.Rel(base, target); back == ".." || strings.HasPrefix(back, ".."+string(os.PathSeparator)) {
		return "", codedError("path_outside_content", nil)
	}
	return target, nil
}

func splitDocument(source string) (front, body, delimiter string) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	lines := strings.Split(source, "\n")
	if len(lines) < 2 || (lines[0] != "---" && lines[0] != "+++") {
		return "", source, ""
	}
	delimiter = lines[0]
	for index := 1; index < len(lines); index++ {
		if lines[index] == delimiter {
			body = strings.Join(lines[index+1:], "\n")
			// The first blank line separates front matter from the body and is
			// not part of the editable Markdown. Any additional blank lines are.
			body = strings.TrimPrefix(body, "\n")
			return strings.Join(lines[1:index], "\n"), body, delimiter
		}
	}
	return "", source, ""
}

func field(front, key, fallback string) string {
	for _, line := range strings.Split(front, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(line, "=", 2)
		}
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), key) {
			return strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		}
	}
	return fallback
}
