package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func writeStartupTrace(output io.Writer, address, site string) {
	if site == "" {
		site = "(not selected)"
	}
	fmt.Fprintf(output, "Started: %s\n", logTimestamp(time.Now()))
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
			duration := formatLogDuration(time.Since(started))
			timestamp := logTimestamp(time.Now())
			outputMu.Lock()
			defer outputMu.Unlock()
			if panicValue != nil {
				fmt.Fprintf(output, "%s [%s] PANIC %v\n%s", timestamp, requestID, panicValue, debug.Stack())
			}
			fmt.Fprintf(output, "%s [%s] %s %s %d %s", timestamp, requestID, r.Method, r.URL.RequestURI(), status, duration)
			if writer.err != nil {
				fmt.Fprintf(output, " error=%q", writer.err.Error())
			}
			fmt.Fprintln(output)
		}()
		next.ServeHTTP(writer, r)
	})
}

func logTimestamp(value time.Time) string {
	return value.Format(time.RFC3339)
}

func formatLogDuration(value time.Duration) string {
	return fmt.Sprintf("%.3fms", float64(value)/float64(time.Millisecond))
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
