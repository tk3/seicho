package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStaticServesEditorRoutes(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest("GET", "/edit/posts/hello.md", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "<title>Seicho</title>") {
		t.Fatal("editor route did not serve the application shell")
	}
	if !strings.Contains(recorder.Body.String(), `<img class="mark" src="/favicon.svg" alt="">`) {
		t.Fatal("application shell does not use the favicon as its header icon")
	}
	if !strings.Contains(recorder.Body.String(), `placeholder="/path/to/hugo-site"`) {
		t.Fatal("site path placeholder is not platform-neutral")
	}
	shell := recorder.Body.String()
	themeIndex := strings.Index(shell, `id="theme-toggle"`)
	siteIndex := strings.Index(shell, `id="change-site"`)
	toolsIndex := strings.Index(shell, `id="tools-menu"`)
	languageIndex := strings.Index(shell, `id="language"`)
	if siteIndex < 0 || themeIndex < siteIndex || toolsIndex < themeIndex || languageIndex < toolsIndex {
		t.Fatal("header controls are not ordered as site change, appearance, and tools")
	}
	if !strings.Contains(shell, `<div class="header-primary"><div class="brand">`) || !strings.Contains(shell, `</div><button id="change-site"`) {
		t.Fatal("site identity and site-change action are not grouped on the left")
	}
	if !strings.Contains(shell, `<summary data-i18n-aria="tools" data-i18n-title="tools">…</summary>`) {
		t.Fatal("tools menu does not use the transparent ellipsis trigger")
	}
	sortIndex := strings.Index(shell, `id="sort"`)
	newPostIndex := strings.Index(shell, `id="new"`)
	if sortIndex < 0 || newPostIndex < sortIndex {
		t.Fatal("sidebar actions are not ordered as sort and new post")
	}
	for _, asset := range []string{"/favicon.svg", "/tokens.css", "/interface-theme.css", "/component-theme.css", "/color-schemes.css", "/router.js", "/i18n.js", "/editor-view.js", "/git-panel.css", "/git-panel.js"} {
		if !strings.Contains(recorder.Body.String(), asset) {
			t.Errorf("application shell does not reference %s", asset)
		}
	}
}

func TestStaticServesRefactoredAssets(t *testing.T) {
	for _, path := range []string{"/favicon.svg", "/tokens.css", "/interface-theme.css", "/component-theme.css", "/color-schemes.css", "/router.js", "/i18n.js", "/editor-view.js", "/git-panel.css", "/git-panel.js"} {
		recorder := httptest.NewRecorder()
		static(recorder, httptest.NewRequest("GET", path, nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d", path, recorder.Code)
		}
	}
}

func TestButtonSizesUseSharedTokens(t *testing.T) {
	tokens := httptest.NewRecorder()
	static(tokens, httptest.NewRequest(http.MethodGet, "/tokens.css", nil))
	for _, declaration := range []string{"--button-height:36px", "--button-height-compact:34px", "--button-height-icon:34px"} {
		if !strings.Contains(tokens.Body.String(), declaration) {
			t.Errorf("tokens.css does not define %s", declaration)
		}
	}

	components := httptest.NewRecorder()
	static(components, httptest.NewRequest(http.MethodGet, "/component-theme.css", nil))
	for _, class := range []string{".button-compact", ".button-icon"} {
		if !strings.Contains(components.Body.String(), class) {
			t.Errorf("component-theme.css does not define %s", class)
		}
	}
}

func TestEditorLayoutKeepsPreviewScrollable(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/interface-theme.css", nil))
	stylesheet := recorder.Body.String()
	for _, rule := range []string{
		`.editor .workspace{height:auto;min-height:0;flex:1}`,
		`.editor .workspace section{min-height:0}`,
		`#body,#preview{padding:22px 26px;font-family:var(--font-mono);font-size:var(--font-size-body);line-height:1.75}`,
		`#preview{min-height:0;flex:1}`,
		`#preview>:first-child{margin-top:0}`,
	} {
		if !strings.Contains(stylesheet, rule) {
			t.Errorf("interface-theme.css does not preserve scroll constraint %s", rule)
		}
	}
}

func TestFaviconUsesApplicationAccentColor(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/favicon.svg", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `fill="#da5b3a"`) {
		t.Fatalf("favicon does not use the application accent color: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
