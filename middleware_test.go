package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAccessTraceWritesRelativeURLAndStatus(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/posts?q=draft", nil))
	if got := output.String(); !strings.Contains(got, "] GET /api/posts?q=draft 201 ") {
		t.Fatalf("unexpected trace: %q", got)
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2}) \[`).MatchString(output.String()) {
		t.Fatalf("trace does not start with an RFC 3339 timestamp: %q", output.String())
	}
	if !regexp.MustCompile(` \d+\.\d{3}ms\n$`).MatchString(output.String()) {
		t.Fatalf("trace duration is not expressed consistently in milliseconds: %q", output.String())
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID was not set")
	}
}

func TestAccessTraceDefaultsStatusToOK(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got := output.String(); !strings.Contains(got, "] GET / 200 ") {
		t.Fatalf("unexpected trace: %q", got)
	}
}

func TestAccessTraceIncludesAPIError(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusInternalServerError, codedError("write_post_failed", errors.New("disk write failed")))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PUT", "/api/post", nil))
	if got := output.String(); !strings.Contains(got, `] PUT /api/post 500 `) || !strings.Contains(got, `error="write_post_failed: disk write failed"`) {
		t.Fatalf("unexpected error trace: %q", got)
	}
}

func TestAPIErrorUsesRequestedLanguage(t *testing.T) {
	handler := languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusBadRequest, codedError("invalid_post_path", nil))
	}))
	req := httptest.NewRequest("GET", "/api/post", nil)
	req.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if got := recorder.Body.String(); !strings.Contains(got, `"code":"invalid_post_path"`) || !strings.Contains(got, "The post path is invalid.") {
		t.Fatalf("error was not translated: %s", got)
	}
}

func TestAPIErrorDefaultsToJapanese(t *testing.T) {
	handler := languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusBadRequest, codedError("invalid_post_path", nil))
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/post", nil))
	if got := recorder.Body.String(); !strings.Contains(got, "投稿パスが不正です") {
		t.Fatalf("unexpected default language: %s", got)
	}
}

func TestLocalizedAPIErrorPreservesTraceMetadata(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, languageResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusInternalServerError, codedError("write_post_failed", errors.New("disk full")))
	})))
	req := httptest.NewRequest("PUT", "/api/post", nil)
	req.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if got := recorder.Body.String(); !strings.Contains(got, `"code":"write_post_failed"`) || !strings.Contains(got, `"requestId":`) {
		t.Fatalf("response metadata is incomplete: %s", got)
	}
	if got := output.String(); !strings.Contains(got, `error="write_post_failed: disk full"`) {
		t.Fatalf("trace error is incomplete: %s", got)
	}
}

func TestAccessTraceRecoversPanic(t *testing.T) {
	var output bytes.Buffer
	handler := accessTrace(&output, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := output.String(); !strings.Contains(got, "PANIC boom") || !strings.Contains(got, "TestAccessTraceRecoversPanic") || !strings.Contains(got, `error="panic: boom"`) {
		t.Fatalf("panic trace is incomplete: %q", got)
	}
}

func TestStartupTraceIncludesRuntimeInformation(t *testing.T) {
	var output bytes.Buffer
	writeStartupTrace(&output, "127.0.0.1:1221", "/path/to/site")
	for _, expected := range []string{
		"Started: ",
		"Seicho " + version,
		"OS: " + runtime.GOOS + "/" + runtime.GOARCH,
		"Go: " + runtime.Version(),
		"Listen: http://127.0.0.1:1221",
		"Site: /path/to/site",
		"Trace: enabled",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("startup trace does not contain %q: %s", expected, output.String())
		}
	}
	firstLine := strings.SplitN(output.String(), "\n", 2)[0]
	if _, err := time.Parse(time.RFC3339, strings.TrimPrefix(firstLine, "Started: ")); err != nil {
		t.Fatalf("startup timestamp is not RFC 3339: %q", firstLine)
	}
}

func TestLogTimestampIncludesDateAndTimezone(t *testing.T) {
	zone := time.FixedZone("JST", 9*60*60)
	if got := logTimestamp(time.Date(2026, 7, 29, 14, 32, 10, 0, zone)); got != "2026-07-29T14:32:10+09:00" {
		t.Fatalf("logTimestamp() = %q", got)
	}
}

func TestLogDurationAlwaysUsesMilliseconds(t *testing.T) {
	for _, test := range []struct {
		value time.Duration
		want  string
	}{
		{420 * time.Microsecond, "0.420ms"},
		{1200 * time.Microsecond, "1.200ms"},
		{2*time.Second + 345*time.Microsecond, "2000.345ms"},
	} {
		if got := formatLogDuration(test.value); got != test.want {
			t.Errorf("formatLogDuration(%s) = %q, want %q", test.value, got, test.want)
		}
	}
}
