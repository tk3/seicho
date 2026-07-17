# Seicho

Seicho is a lightweight, local-only editor for managing files in a Hugo site's `content` directory from your browser.

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Getting started

```bash
go run . -port 1221
```

Open <http://127.0.0.1:1221> in your browser and enter the absolute path to your Hugo site. You can also specify the site when starting Seicho.

```bash
go run . -site /path/to/hugo-site
```

To build an executable:

```bash
go build -buildvcs=false -o seicho .
```

Run the built executable from Git Bash:

```bash
./seicho -port 1221
```

To specify a Hugo site at startup:

```bash
./seicho -site /path/to/hugo-site
```

Use `-port` to change the port. The default is `1221`.

```bash
./seicho -port 8080
```

You can specify the site and port together:

```bash
./seicho -site /path/to/hugo-site -port 8080
```

To display version information:

```bash
./seicho -version
```

Use `-trace` to write startup information and access details to standard output. Access logs include a request ID, HTTP method, relative URL, HTTP status, processing time, and any API error. If a panic occurs, Seicho also writes a stack trace with the same request ID.

```bash
./seicho -port 1221 -trace
```

Example output:

```text
Seicho 0.2.7
OS: windows/amd64
Go: go1.26.5
PID: 12345
Listen: http://127.0.0.1:1221
Site: /path/to/hugo-site
Trace: enabled
[00000001] GET / 200 420µs
[00000002] GET /api/posts 200 1.2ms
[00000003] PUT /api/post 500 2.1ms error="open content/posts/example.md: permission denied"
```

## Features

- List, search, and sort Markdown posts by modification time
- Create posts with `hugo new content`, including support for `archetypes/default.md`
- Edit content while preserving YAML or TOML front matter
- Create, save, rename, and delete posts
- Live Markdown preview using Goldmark, the same parser used by Hugo
- Detect conflicting external file changes before overwriting
- Prevent file operations outside the site's `content` directory

Starting a Hugo preview server, uploading images, and performing Git operations are not currently supported.

The live preview uses the same Goldmark parser as Hugo. It supports CommonMark, tables, strikethrough, task lists, definition lists, footnotes, and more. Reproducing the final site output—including shortcodes, themes, and render hooks—will require the planned Hugo preview server integration.

The `hugo` command must be available on your `PATH` to create new posts. Seicho runs the following command with the selected site as its working directory:

```bash
hugo new content posts/example.md
```

## License

Licensed under the [Apache License 2.0](LICENSE).
