package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed web/*
var webFiles embed.FS

// version can be replaced at build time with:
// go build -ldflags "-X main.version=1.0.0" .
var version = "0.1.0"

type server struct {
	mu       sync.RWMutex
	root     string
	markdown goldmark.Markdown
}

type postSummary struct {
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Date     string    `json:"date"`
	Draft    bool      `json:"draft"`
	Modified time.Time `json:"modified"`
}

type postDocument struct {
	Path        string `json:"path"`
	FrontMatter string `json:"frontMatter"`
	Body        string `json:"body"`
	Delimiter   string `json:"delimiter,omitempty"`
	Modified    string `json:"modified,omitempty"`
}

func main() {
	root := flag.String("site", "", "Hugo site directory")
	port := flag.Int("port", 1314, "listen port")
	addr := flag.String("addr", "", "listen address (overrides -port)")
	showVersion := flag.Bool("version", false, "show version")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		command := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
		fmt.Fprintf(out, "Seicho %s - local editor for Hugo posts\n\n", version)
		fmt.Fprintf(out, "Usage:\n  %s [options]\n\nOptions:\n", command)
		flag.PrintDefaults()
		fmt.Fprintln(out, "\nExamples:")
		fmt.Fprintf(out, "  %s -port 1314\n", command)
		fmt.Fprintf(out, "  %s -site /path/to/hugo-site -port 8080\n", command)
	}
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	flag.Parse()
	if *showVersion {
		fmt.Printf("Seicho %s\n", version)
		return
	}
	if *port < 1 || *port > 65535 {
		log.Fatal("port must be between 1 and 65535")
	}
	listenAddress := *addr
	if listenAddress == "" {
		listenAddress = fmt.Sprintf("127.0.0.1:%d", *port)
	}
	s := &server{markdown: newMarkdownRenderer()}
	if *root != "" {
		if err := s.setRoot(*root); err != nil {
			log.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/site", s.site)
	mux.HandleFunc("/api/posts", s.posts)
	mux.HandleFunc("/api/post", s.post)
	mux.HandleFunc("/api/preview", s.preview)
	mux.HandleFunc("/", static)
	ln, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Seicho: http://%s\n", ln.Addr())
	log.Fatal(http.Serve(ln, securityHeaders(mux)))
}

func newMarkdownRenderer() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
	))
}

func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Markdown string `json:"markdown"`
	}
	if err := decodeJSON(r, &req); err != nil {
		apiError(w, 400, err)
		return
	}
	renderer := s.markdown
	if renderer == nil {
		renderer = newMarkdownRenderer()
	}
	var output bytes.Buffer
	if err := renderer.Convert([]byte(req.Markdown), &output); err != nil {
		apiError(w, 500, err)
		return
	}
	jsonResponse(w, 200, map[string]string{"html": output.String()})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'")
		next.ServeHTTP(w, r)
	})
}

func static(w http.ResponseWriter, r *http.Request) {
	name := "web" + r.URL.Path
	if r.URL.Path == "/" {
		name = "web/index.html"
	}
	b, err := webFiles.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(name)))
	w.Write(b)
}

func (s *server) setRoot(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return errors.New("指定したフォルダーが見つかりません")
	}
	if !isHugoSite(abs) {
		return errors.New("Hugoサイト設定ファイルが見つかりません")
	}
	if err := os.MkdirAll(filepath.Join(abs, "content"), 0755); err != nil {
		return err
	}
	s.mu.Lock()
	s.root = abs
	s.mu.Unlock()
	return nil
}

func isHugoSite(root string) bool {
	for _, n := range []string{"hugo.toml", "hugo.yaml", "hugo.yml", "config.toml", "config.yaml", "config.yml"} {
		if _, err := os.Stat(filepath.Join(root, n)); err == nil {
			return true
		}
	}
	return false
}

func (s *server) getRoot() string { s.mu.RLock(); defer s.mu.RUnlock(); return s.root }

func (s *server) site(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		jsonResponse(w, 200, map[string]any{"path": s.getRoot(), "configured": s.getRoot() != "", "version": version})
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
		apiError(w, 400, err)
		return
	}
	if err := s.setRoot(req.Path); err != nil {
		apiError(w, 400, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"path": s.getRoot(), "configured": true, "version": version})
}

func (s *server) posts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	root := s.getRoot()
	if root == "" {
		apiError(w, 409, errors.New("Hugoサイトを選択してください"))
		return
	}
	items := []postSummary{}
	err := filepath.WalkDir(filepath.Join(root, "content"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		front, _, _ := splitDocument(string(data))
		info, _ := d.Info()
		rel, _ := filepath.Rel(filepath.Join(root, "content"), path)
		items = append(items, postSummary{Path: filepath.ToSlash(rel), Title: field(front, "title", strings.TrimSuffix(d.Name(), ext)), Date: field(front, "date", ""), Draft: strings.EqualFold(field(front, "draft", "false"), "true"), Modified: info.ModTime()})
		return nil
	})
	if err != nil {
		apiError(w, 500, err)
		return
	}
	jsonResponse(w, 200, items)
}

func (s *server) post(w http.ResponseWriter, r *http.Request) {
	root := s.getRoot()
	if root == "" {
		apiError(w, 409, errors.New("Hugoサイトを選択してください"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		path, err := safePostPath(root, r.URL.Query().Get("path"))
		if err != nil {
			apiError(w, 400, err)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			apiError(w, 404, errors.New("投稿が見つかりません"))
			return
		}
		info, _ := os.Stat(path)
		front, body, delim := splitDocument(string(data))
		rel, _ := filepath.Rel(filepath.Join(root, "content"), path)
		jsonResponse(w, 200, postDocument{Path: filepath.ToSlash(rel), FrontMatter: front, Body: body, Delimiter: delim, Modified: fileVersion(info)})
	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := decodeJSON(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		path, err := safePostPath(root, req.Path)
		if err != nil {
			apiError(w, 400, err)
			return
		}
		if _, err := os.Stat(path); err == nil {
			apiError(w, 409, errors.New("同じパスの投稿がすでに存在します"))
			return
		}
		rel, _ := filepath.Rel(filepath.Join(root, "content"), path)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "hugo", "new", "content", filepath.ToSlash(rel))
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				apiError(w, 504, errors.New("hugo newがタイムアウトしました"))
				return
			}
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			apiError(w, 500, fmt.Errorf("hugo newの実行に失敗しました: %s", message))
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			apiError(w, 500, errors.New("hugo newは成功しましたが、生成ファイルを読み込めません"))
			return
		}
		info, err := os.Stat(path)
		if err != nil {
			apiError(w, 500, err)
			return
		}
		front, body, delim := splitDocument(string(data))
		jsonResponse(w, http.StatusCreated, postDocument{Path: filepath.ToSlash(rel), FrontMatter: front, Body: body, Delimiter: delim, Modified: fileVersion(info)})
	case http.MethodPut:
		var doc postDocument
		if err := decodeJSON(r, &doc); err != nil {
			apiError(w, 400, err)
			return
		}
		path, err := safePostPath(root, doc.Path)
		if err != nil {
			apiError(w, 400, err)
			return
		}
		if info, err := os.Stat(path); err == nil && doc.Modified != "" && fileVersion(info) != doc.Modified {
			apiError(w, 409, errors.New("保存後にファイルが変更されています。再読み込みしてください"))
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			apiError(w, 500, err)
			return
		}
		delim := doc.Delimiter
		if delim != "+++" {
			delim = "---"
		}
		content := delim + "\n" + strings.TrimSpace(doc.FrontMatter) + "\n" + delim + "\n\n" + doc.Body
		tmp := path + ".seicho-tmp"
		if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
			apiError(w, 500, err)
			return
		}
		if err := os.Rename(tmp, path); err != nil {
			os.Remove(tmp)
			apiError(w, 500, err)
			return
		}
		saved, err := os.ReadFile(path)
		if err != nil {
			apiError(w, 500, fmt.Errorf("保存後のファイルを確認できません: %w", err))
			return
		}
		if !bytes.Equal(saved, []byte(content)) {
			apiError(w, 500, errors.New("保存後のファイル内容が一致しません"))
			return
		}
		info, _ := os.Stat(path)
		doc.Modified = fileVersion(info)
		jsonResponse(w, 200, doc)
	case http.MethodDelete:
		path, err := safePostPath(root, r.URL.Query().Get("path"))
		if err != nil {
			apiError(w, 400, err)
			return
		}
		if err := os.Remove(path); err != nil {
			apiError(w, 404, errors.New("投稿が見つかりません"))
			return
		}
		w.WriteHeader(204)
	default:
		methodNotAllowed(w)
	}
}

func fileVersion(info os.FileInfo) string {
	return info.ModTime().UTC().Format(time.RFC3339Nano)
}

func safePostPath(root, rel string) (string, error) {
	rel = filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel)))
	if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return "", errors.New("投稿パスが不正です")
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != ".md" && ext != ".markdown" {
		return "", errors.New("拡張子は .md または .markdown にしてください")
	}
	base := filepath.Join(root, "content")
	target := filepath.Join(base, rel)
	if back, _ := filepath.Rel(base, target); back == ".." || strings.HasPrefix(back, ".."+string(os.PathSeparator)) {
		return "", errors.New("contentフォルダー外は操作できません")
	}
	return target, nil
}

func splitDocument(src string) (front, body, delimiter string) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(src, "\n")
	if len(lines) < 2 || (lines[0] != "---" && lines[0] != "+++") {
		return "", src, ""
	}
	delim := lines[0]
	for i := 1; i < len(lines); i++ {
		if lines[i] == delim {
			body := strings.Join(lines[i+1:], "\n")
			// The first blank line separates front matter from the body and is
			// not part of the editable Markdown. Any additional blank lines are.
			body = strings.TrimPrefix(body, "\n")
			return strings.Join(lines[1:i], "\n"), body, delim
		}
	}
	return "", src, ""
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

func decodeJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 4<<20)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func apiError(w http.ResponseWriter, status int, err error) {
	jsonResponse(w, status, map[string]string{"error": err.Error()})
}
func methodNotAllowed(w http.ResponseWriter) {
	apiError(w, 405, errors.New("許可されていない操作です"))
}
