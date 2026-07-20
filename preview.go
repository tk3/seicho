package main

import (
	"bytes"
	"net/http"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

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
		apiError(w, http.StatusBadRequest, err)
		return
	}
	renderer := s.markdown
	if renderer == nil {
		renderer = newMarkdownRenderer()
	}
	var output bytes.Buffer
	if err := renderer.Convert([]byte(req.Markdown), &output); err != nil {
		apiError(w, http.StatusInternalServerError, codedError("preview_failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"html": output.String()})
}
