package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed web/*
var webFiles embed.FS

// version can be replaced at build time with:
// go build -ldflags "-X main.version=1.0.0" .
var version = "0.2.13"

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
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath,omitempty"`
	FrontMatter  string `json:"frontMatter"`
	Body         string `json:"body"`
	Delimiter    string `json:"delimiter,omitempty"`
	Modified     string `json:"modified,omitempty"`
}

func main() {
	root := flag.String("site", "", "Hugo site directory")
	port := flag.Int("port", 1221, "listen port")
	addr := flag.String("addr", "", "listen address (overrides -port)")
	trace := flag.Bool("trace", false, "write access logs to stdout")
	showVersion := flag.Bool("version", false, "show version")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		command := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
		fmt.Fprintf(out, "Seicho %s - local editor for Hugo posts\n\n", version)
		fmt.Fprintf(out, "Usage:\n  %s [options]\n\nOptions:\n", command)
		flag.PrintDefaults()
		fmt.Fprintln(out, "\nExamples:")
		fmt.Fprintf(out, "  %s -port 1221\n", command)
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
	var handler http.Handler = securityHeaders(languageResponses(mux))
	if *trace {
		writeStartupTrace(os.Stdout, ln.Addr().String(), s.getRoot())
		handler = accessTrace(os.Stdout, handler)
	} else {
		fmt.Printf("Seicho: http://%s\n", ln.Addr())
	}
	log.Fatal(http.Serve(ln, handler))
}

func writeStartupTrace(output io.Writer, address, site string) {
	if site == "" {
		site = "(not selected)"
	}
	fmt.Fprintf(output, "Seicho %s\n", version)
	fmt.Fprintf(output, "OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(output, "Go: %s\n", runtime.Version())
	fmt.Fprintf(output, "PID: %d\n", os.Getpid())
	fmt.Fprintf(output, "Listen: http://%s\n", address)
	fmt.Fprintf(output, "Site: %s\n", site)
	fmt.Fprintln(output, "Trace: enabled")
}

type statusWriter struct {
	http.ResponseWriter
	status    int
	requestID string
	err       error
}

func (w *statusWriter) recordError(err error)  { w.err = err }
func (w *statusWriter) traceRequestID() string { return w.requestID }
func (w *statusWriter) responseLanguage() string {
	if localized, ok := w.ResponseWriter.(interface{ responseLanguage() string }); ok {
		return localized.responseLanguage()
	}
	return "ja"
}

var requestSequence uint64

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func accessTrace(output io.Writer, next http.Handler) http.Handler {
	var outputMu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := fmt.Sprintf("%08x", atomic.AddUint64(&requestSequence, 1))
		writer := &statusWriter{ResponseWriter: w, requestID: requestID}
		writer.Header().Set("X-Request-ID", requestID)
		defer func() {
			panicValue := recover()
			if panicValue != nil {
				if writer.status == 0 {
					apiError(writer, http.StatusInternalServerError, codedError("internal_error", errors.New("unexpected panic")))
				} else {
					writer.status = http.StatusInternalServerError
				}
				writer.err = fmt.Errorf("panic: %v", panicValue)
			}
			status := writer.status
			if status == 0 {
				status = http.StatusOK
			}
			duration := time.Since(started).Round(time.Microsecond)
			outputMu.Lock()
			defer outputMu.Unlock()
			if panicValue != nil {
				fmt.Fprintf(output, "[%s] PANIC %v\n%s", requestID, panicValue, debug.Stack())
			}
			fmt.Fprintf(output, "[%s] %s %s %d %s", requestID, r.Method, r.URL.RequestURI(), status, duration)
			if writer.err != nil {
				fmt.Fprintf(output, " error=%q", writer.err.Error())
			}
			fmt.Fprintln(output)
		}()
		next.ServeHTTP(writer, r)
	})
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
		apiError(w, 500, codedError("preview_failed", err))
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

type languageWriter struct {
	http.ResponseWriter
	language string
}

func (w *languageWriter) responseLanguage() string { return w.language }
func (w *languageWriter) recordError(err error) {
	if recorder, ok := w.ResponseWriter.(interface{ recordError(error) }); ok {
		recorder.recordError(err)
	}
}
func (w *languageWriter) traceRequestID() string {
	if traced, ok := w.ResponseWriter.(interface{ traceRequestID() string }); ok {
		return traced.traceRequestID()
	}
	return ""
}

func languageResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		language := "ja"
		if strings.HasPrefix(strings.ToLower(r.Header.Get("Accept-Language")), "en") {
			language = "en"
		}
		next.ServeHTTP(&languageWriter{ResponseWriter: w, language: language}, r)
	})
}

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
	w.Write(b)
}

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
		apiError(w, 409, codedError("site_not_selected", nil))
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
		apiError(w, 500, codedError("list_posts_failed", err))
		return
	}
	jsonResponse(w, 200, items)
}

func (s *server) post(w http.ResponseWriter, r *http.Request) {
	root := s.getRoot()
	if root == "" {
		apiError(w, 409, codedError("site_not_selected", nil))
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
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			apiError(w, status, codedError("read_post_failed", err))
			return
		}
		info, err := os.Stat(path)
		if err != nil {
			apiError(w, 500, codedError("read_file_info_failed", err))
			return
		}
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
			apiError(w, 409, codedError("post_already_exists", nil))
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
				apiError(w, 504, codedError("hugo_new_timeout", ctx.Err()))
				return
			}
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			apiError(w, 500, codedError("hugo_new_failed", errors.New(message)))
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			apiError(w, 500, codedError("read_generated_post_failed", err))
			return
		}
		info, err := os.Stat(path)
		if err != nil {
			apiError(w, 500, codedError("read_file_info_failed", err))
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
		originalPath := path
		if doc.OriginalPath != "" {
			originalPath, err = safePostPath(root, doc.OriginalPath)
			if err != nil {
				apiError(w, 400, err)
				return
			}
		}
		renaming := filepath.Clean(originalPath) != filepath.Clean(path)
		if info, err := os.Stat(originalPath); err == nil && doc.Modified != "" && fileVersion(info) != doc.Modified {
			apiError(w, 409, codedError("file_modified", nil))
			return
		}
		if renaming {
			if _, err := os.Stat(path); err == nil {
				apiError(w, 409, codedError("destination_exists", nil))
				return
			} else if !errors.Is(err, os.ErrNotExist) {
				apiError(w, 500, codedError("inspect_destination_failed", err))
				return
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			apiError(w, 500, codedError("create_post_directory_failed", err))
			return
		}
		delim := doc.Delimiter
		if delim != "+++" {
			delim = "---"
		}
		content := delim + "\n" + strings.TrimSpace(doc.FrontMatter) + "\n" + delim + "\n\n" + doc.Body
		tmp := path + ".seicho-tmp"
		if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
			apiError(w, 500, codedError("write_post_failed", err))
			return
		}
		if err := os.Rename(tmp, path); err != nil {
			os.Remove(tmp)
			apiError(w, 500, codedError("replace_post_failed", err))
			return
		}
		saved, err := os.ReadFile(path)
		if err != nil {
			apiError(w, 500, codedError("verify_saved_post_failed", err))
			return
		}
		if !bytes.Equal(saved, []byte(content)) {
			apiError(w, 500, codedError("saved_content_mismatch", nil))
			return
		}
		if renaming {
			if err := os.Remove(originalPath); err != nil {
				os.Remove(path)
				apiError(w, 500, codedError("remove_original_post_failed", err))
				return
			}
		}
		info, _ := os.Stat(path)
		doc.OriginalPath = ""
		doc.Modified = fileVersion(info)
		jsonResponse(w, 200, doc)
	case http.MethodDelete:
		path, err := safePostPath(root, r.URL.Query().Get("path"))
		if err != nil {
			apiError(w, 400, err)
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
		return "", codedError("invalid_post_path", nil)
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != ".md" && ext != ".markdown" {
		return "", codedError("invalid_post_extension", nil)
	}
	base := filepath.Join(root, "content")
	target := filepath.Join(base, rel)
	if back, _ := filepath.Rel(base, target); back == ".." || strings.HasPrefix(back, ".."+string(os.PathSeparator)) {
		return "", codedError("path_outside_content", nil)
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
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return codedError("invalid_request", err)
	}
	return nil
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func apiError(w http.ResponseWriter, status int, err error) {
	if recorder, ok := w.(interface{ recordError(error) }); ok {
		recorder.recordError(err)
	}
	code := "internal_error"
	var failure *apiFailure
	if errors.As(err, &failure) {
		code = failure.code
	}
	language := "ja"
	if localized, ok := w.(interface{ responseLanguage() string }); ok {
		language = localized.responseLanguage()
	}
	response := map[string]string{"code": code, "error": errorMessage(code, language)}
	if traced, ok := w.(interface{ traceRequestID() string }); ok {
		response["requestId"] = traced.traceRequestID()
	}
	jsonResponse(w, status, response)
}

type apiFailure struct {
	code  string
	cause error
}

func (e *apiFailure) Error() string {
	if e.cause == nil {
		return e.code
	}
	return e.code + ": " + e.cause.Error()
}

func (e *apiFailure) Unwrap() error { return e.cause }

func codedError(code string, cause error) error {
	return &apiFailure{code: code, cause: cause}
}

type localizedMessages struct {
	ja string
	en string
}

var apiErrorMessages = map[string]localizedMessages{
	"internal_error":                  {"サーバー内部で予期しないエラーが発生しました。", "An unexpected internal server error occurred."},
	"invalid_request":                 {"リクエストの内容が正しくありません。", "The request is invalid."},
	"preview_failed":                  {"Markdownをプレビューできません。", "Could not preview the Markdown."},
	"invalid_site_path":               {"サイトフォルダーのパスが正しくありません。", "The site folder path is invalid."},
	"site_folder_not_found":           {"指定したフォルダーが見つかりません。", "The specified folder was not found."},
	"hugo_config_not_found":           {"Hugoサイト設定ファイルが見つかりません。", "No Hugo site configuration file was found."},
	"create_content_directory_failed": {"contentフォルダーを作成できません。", "Could not create the content folder."},
	"site_not_selected":               {"Hugoサイトを選択してください。", "Select a Hugo site first."},
	"list_posts_failed":               {"投稿の一覧を読み込めません。", "Could not load the post list."},
	"read_post_failed":                {"投稿ファイルを読み込めません。", "Could not read the post file."},
	"read_file_info_failed":           {"投稿ファイルの情報を取得できません。", "Could not read the post file information."},
	"post_already_exists":             {"同じパスの投稿がすでに存在します。", "A post already exists at the same path."},
	"hugo_new_timeout":                {"hugo newがタイムアウトしました。", "hugo new timed out."},
	"hugo_new_failed":                 {"hugo newの実行に失敗しました。", "Failed to run hugo new."},
	"read_generated_post_failed":      {"hugo newは成功しましたが、生成ファイルを読み込めません。", "hugo new succeeded, but the generated file could not be read."},
	"file_modified":                   {"保存後にファイルが変更されています。再読み込みしてください。", "The file has been modified since it was loaded. Reload it and try again."},
	"destination_exists":              {"変更先のパスには既に投稿が存在します。", "A post already exists at the destination path."},
	"inspect_destination_failed":      {"変更先のファイルを確認できません。", "Could not inspect the destination file."},
	"create_post_directory_failed":    {"投稿先のフォルダーを作成できません。", "Could not create the post folder."},
	"write_post_failed":               {"投稿ファイルを書き込めません。", "Could not write the post file."},
	"replace_post_failed":             {"投稿ファイルを保存できません。", "Could not save the post file."},
	"verify_saved_post_failed":        {"保存後のファイルを確認できません。", "Could not verify the saved file."},
	"saved_content_mismatch":          {"保存後のファイル内容が一致しません。", "The saved file content does not match the requested content."},
	"remove_original_post_failed":     {"元ファイルを削除できないため名前を変更できません。", "Could not rename the post because the original file could not be removed."},
	"delete_post_failed":              {"投稿ファイルを削除できません。", "Could not delete the post file."},
	"invalid_post_path":               {"投稿パスが不正です。", "The post path is invalid."},
	"invalid_post_extension":          {"拡張子は .md または .markdown にしてください。", "Use the .md or .markdown file extension."},
	"path_outside_content":            {"contentフォルダー外は操作できません。", "Files outside the content folder cannot be accessed."},
	"method_not_allowed":              {"許可されていない操作です。", "This operation is not allowed."},
}

func errorMessage(code, language string) string {
	messages, ok := apiErrorMessages[code]
	if !ok {
		messages = apiErrorMessages["internal_error"]
	}
	if language == "en" {
		return messages.en
	}
	return messages.ja
}
func methodNotAllowed(w http.ResponseWriter) {
	apiError(w, 405, codedError("method_not_allowed", nil))
}
