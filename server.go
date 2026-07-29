package main

import (
	"context"
	"sync"
	"time"

	"github.com/yuin/goldmark"
)

type server struct {
	mu       sync.RWMutex
	root     string
	markdown goldmark.Markdown
	hugoNew  func(context.Context, string, string) ([]byte, error)
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
