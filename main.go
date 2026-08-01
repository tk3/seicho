package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// version can be replaced at build time with:
// go build -ldflags "-X main.version=1.0.0" .
var version = "0.2.24"

func main() {
	root := flag.String("site", "", "Hugo site directory")
	port := flag.Int("port", 1221, "listen port")
	addr := flag.String("addr", "", "listen address (overrides -port)")
	trace := flag.Bool("trace", false, "write access logs to stdout")
	showVersion := flag.Bool("version", false, "show version")
	flag.Usage = func() {
		output := flag.CommandLine.Output()
		command := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
		fmt.Fprintf(output, "Seicho %s - local editor for Hugo posts\n\n", version)
		fmt.Fprintf(output, "Usage:\n  %s [options]\n\nOptions:\n", command)
		flag.PrintDefaults()
		fmt.Fprintln(output, "\nExamples:")
		fmt.Fprintf(output, "  %s -port 1221\n", command)
		fmt.Fprintf(output, "  %s -site /path/to/hugo-site -port 8080\n", command)
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

	server := &server{markdown: newMarkdownRenderer()}
	if *root != "" {
		if err := server.setRoot(*root); err != nil {
			log.Fatal(err)
		}
	}

	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatal(err)
	}
	handler := server.handler()
	if *trace {
		writeStartupTrace(os.Stdout, listener.Addr().String(), server.getRoot())
		handler = accessTrace(os.Stdout, handler)
	} else {
		fmt.Printf("Seicho: http://%s\n", listener.Addr())
	}
	log.Fatal(http.Serve(listener, handler))
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/site", s.site)
	mux.HandleFunc("/api/posts", s.posts)
	mux.HandleFunc("/api/post", s.post)
	mux.HandleFunc("/api/preview", s.preview)
	mux.HandleFunc("/api/git/status", s.gitStatus)
	mux.HandleFunc("/api/git/diff", s.gitDiff)
	mux.HandleFunc("/api/git/stage", s.gitStage)
	mux.HandleFunc("/api/git/commit", s.gitCommit)
	mux.HandleFunc("/", static)
	return securityHeaders(languageResponses(mux))
}
