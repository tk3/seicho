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

func TestHeaderHeightUsesSharedToken(t *testing.T) {
	tokens := httptest.NewRecorder()
	static(tokens, httptest.NewRequest(http.MethodGet, "/tokens.css", nil))
	if !strings.Contains(tokens.Body.String(), `--header-height:56px`) {
		t.Fatal("tokens.css does not define the compact header height")
	}

	interfaceStyles := httptest.NewRecorder()
	static(interfaceStyles, httptest.NewRequest(http.MethodGet, "/interface-theme.css", nil))
	if !strings.Contains(interfaceStyles.Body.String(), `grid-template-rows:var(--header-height) calc(100vh - var(--header-height))`) {
		t.Fatal("application shell does not use the shared header height")
	}

	componentStyles := httptest.NewRecorder()
	static(componentStyles, httptest.NewRequest(http.MethodGet, "/component-theme.css", nil))
	if !strings.Contains(componentStyles.Body.String(), `.git-panel{top:var(--header-height)`) {
		t.Fatal("Git panel is not aligned to the shared header height")
	}
}

func TestHeaderIdentityUsesConsistentAlignment(t *testing.T) {
	shell := httptest.NewRecorder()
	static(shell, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(shell.Body.String(), `<div class="brand-identity"><strong class="brand-name">Seicho</strong><span id="app-version"></span><small id="site-name"></small></div>`) {
		t.Fatal("header identity is not composed from independent aligned elements")
	}

	styles := httptest.NewRecorder()
	static(styles, httptest.NewRequest(http.MethodGet, "/interface-theme.css", nil))
	stylesheet := styles.Body.String()
	for _, rule := range []string{
		`.header-primary{gap:6px}`,
		`.brand>.brand-identity{display:flex;min-width:0;flex-direction:row;align-items:center;gap:8px}`,
		`.brand-name,#app-version,.brand small{line-height:20px}`,
		`.brand small{max-width:420px;margin-left:24px`,
	} {
		if !strings.Contains(stylesheet, rule) {
			t.Errorf("interface-theme.css does not preserve header alignment rule %s", rule)
		}
	}
	if strings.Contains(stylesheet, `vertical-align:middle`) {
		t.Fatal("application version retains an independent vertical alignment")
	}
}

func TestContentsPostTitlesAreEmphasized(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/interface-theme.css", nil))
	if !strings.Contains(recorder.Body.String(), `.post strong{color:#24292f;font-size:var(--font-size-compact);font-weight:600}`) {
		t.Fatal("Contents post titles do not use the emphasized font weight")
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

func TestZenModeProvidesThemeToggle(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(recorder.Body.String(), `id="zen-theme-toggle" class="theme-toggle zen-theme-toggle"`) {
		t.Fatal("Zen Mode does not provide the shared theme toggle")
	}

	recorder = httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/interface-theme.css", nil))
	stylesheet := recorder.Body.String()
	if !strings.Contains(stylesheet, `.editor-toolbar .zen-theme-toggle{display:none}`) || !strings.Contains(stylesheet, `body.zen-mode .editor-toolbar .zen-theme-toggle{display:grid}`) {
		t.Fatal("Zen Mode theme toggle visibility is not scoped to Zen Mode")
	}

	recorder = httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	script := recorder.Body.String()
	if !strings.Contains(script, `document.querySelectorAll('.theme-toggle')`) || !strings.Contains(script, `$('#theme-toggle').onclick=$('#zen-theme-toggle').onclick=toggleTheme`) {
		t.Fatal("theme controls do not share synchronized behavior")
	}
}

func TestFaviconUsesApplicationAccentColor(t *testing.T) {
	recorder := httptest.NewRecorder()
	static(recorder, httptest.NewRequest(http.MethodGet, "/favicon.svg", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `fill="#da5b3a"`) {
		t.Fatalf("favicon does not use the application accent color: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
