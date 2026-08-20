# webclone

> A fast, from-scratch website mirror tool written in Go — with both a CLI and a browser-based GUI.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](#building-from-source)

`webclone` mirrors an entire website to your local disk. Unlike a single-page
cloner, it follows internal links recursively and saves every page and asset
(HTML, CSS, JS, images, fonts, video/audio, ...) it finds, reproducing the
site's URL structure under the output directory so the mirrored copy can be
browsed **fully offline**.

It ships in two forms:

- **`webclone`** — a command-line tool for scripting and automation.
- **`webclone-gui`** — a browser-based desktop GUI (pure Go, no CGO) with a
  bilingual (Persian / English) dark interface and live progress streaming.

It is a from-scratch rewrite inspired by
[`goclone`](https://github.com/goclone-dev/goclone), with full-site recursive
crawling instead of single-page cloning.

---

## Table of contents

- [Features](#features)
- [Installation](#installation)
- [CLI usage](#cli-usage)
- [Graphical UI](#graphical-ui-gui)
- [Mirror layout](#mirror-layout)
- [Link rewriting](#link-rewriting)
- [Output structure](#output-structure)
- [Building from source](#building-from-source)
- [Project layout](#project-layout)
- [Differences from goclone](#differences-from-goclone)
- [License](#license)

---

## Features

- **Recursive crawling** — follows `<a href>`, `<iframe src>`, and (optionally)
  external links.
- **Mirror layout** — `https://example.com/foo/bar` is saved to
  `example.com/foo/bar/index.html`, so the on-disk tree mirrors the URL
  hierarchy exactly.
- **Link rewriting** — every absolute or root-relative URL inside the
  downloaded HTML/CSS is rewritten to a relative path pointing at the local
  copy, so the mirror works offline with no server-side rewriting.
- **Full asset coverage** — CSS (`url()` and `@import`), JS, images
  (`src` and `srcset`), fonts, video/audio, inline `style="..."`, `<style>`
  blocks, `<source>`, `<embed>`, `<object>`, `<iframe>`, and
  `<meta http-equiv="refresh">`.
- **Concurrency** — bounded worker pool (default 5, configurable).
- **Safety caps** — `--max-urls` and `--max-depth` prevent runaway crawls.
- **Domain restrictions** — `--same-domain` (on by default), `--allow-subdomains`,
  and `--allowed-hosts` give fine-grained control over which hosts to follow.
- **Cookie support** — pre-set cookies with `--cookie` for sites that require a
  session.
- **Proxy + custom UA + insecure TLS** — full HTTP client configuration.
- **Manifest** — `--manifest` (on by default) writes a `manifest.json` listing
  every URL crawled.
- **Built-in file server** — `--serve` starts a static HTTP server so you can
  preview the mirror immediately.
- **Browser-based GUI** — cross-platform, zero external dependencies, live
  progress over Server-Sent Events, bilingual FA/EN with RTL/LTR toggle.

## Installation

From source:

```bash
git clone https://github.com/a-talebifard/webclone.git
cd webclone
go build -o webclone ./cmd/webclone
./webclone --help
```

Or with Go installed:

```bash
go install github.com/a-talebifard/webclone/cmd/webclone@latest
```

## CLI usage

```bash
webclone [flags] <url> [<url2> ...]
```

### Common examples

```bash
# Mirror a site (same-domain by default), all internal pages:
webclone https://example.com

# Mirror to a specific output dir:
webclone -o ./mirror https://example.com

# Limit to 3 link hops from the seed (faster, smaller mirror):
webclone --max-depth 3 https://example.com

# Also follow external links (turn off the same-domain restriction):
webclone --same-domain=false https://example.com

# Custom workers + custom UA + proxy:
webclone -w 10 -u "Mozilla/5.0 ..." --proxy http://localhost:8080 https://example.com

# Explicit host allowlist (overrides --same-domain):
webclone --allowed-hosts example.com,cdn.example.com https://example.com

# Mirror and immediately serve on http://localhost:8080:
webclone -s https://example.com

# Pre-set cookies (e.g. for sites that require a session):
webclone -C "session=abc123; token=xyz" https://example.com
```

### All flags

| Flag                 | Default   | Description                                                          |
| -------------------- | --------- | -------------------------------------------------------------------- |
| `-o, --output`       | `.`       | Output directory for the mirror tree                                 |
| `-w, --workers`      | `5`       | Concurrent fetch workers                                             |
| `--max-urls`         | `10000`   | Hard cap on total URLs fetched (0 = unlimited)                       |
| `--max-depth`        | `0`       | Max link-hops from the seed (0 = unlimited)                          |
| `--same-domain`      | `true`    | Only crawl URLs on the seed's registered domain                      |
| `--allow-subdomains` | `true`    | When `--same-domain` is set, also follow subdomains of the seed host |
| `--allowed-hosts`    | (empty)   | Comma-separated explicit list of hostnames to crawl (overrides `--same-domain`) |
| `--skip-assets`      | `false`   | Only download HTML pages, skip CSS/JS/images                         |
| `--asset-ext`        | (empty)   | Comma-separated list of asset extensions to download (empty = all)   |
| `--timeout`          | `60s`     | Per-request timeout                                                  |
| `-u, --user-agent`   | (default) | Custom User-Agent string                                             |
| `-p, --proxy`        | (empty)   | Proxy URL (`http://`, `https://`, `socks5://`)                       |
| `--insecure-tls`     | `false`   | Skip TLS certificate verification                                    |
| `-v, --verbose`      | `false`   | Verbose per-URL log output                                           |
| `-q, --quiet`        | `false`   | Suppress all non-error output                                        |
| `-s, --serve`        | `false`   | After mirroring, serve the output dir on a local HTTP server         |
| `-P, --serve-port`   | `8080`    | Port for `--serve`                                                   |
| `--manifest`         | `true`    | Write `manifest.json` with all crawled URLs                          |
| `-C, --cookie`       | (empty)   | Pre-set cookies (`name=value; name2=value2`)                         |

## Graphical UI (GUI)

`webclone-gui` is a **browser-based** desktop GUI. It runs a local HTTP server
and opens your default browser — no CGO, no MinGW, no external dependencies,
just pure Go.

### Build and run

```bash
go build -o webclone-gui ./cmd/webclone-gui
./webclone-gui
```

On Windows:

```powershell
go build -o webclone-gui.exe .\cmd\webclone-gui
.\webclone-gui.exe
```

Your browser opens automatically at `http://127.0.0.1:8080/`. To use a custom
port:

```bash
./webclone-gui 9090   # use port 9090 instead of 8080
```

### Why a browser-based GUI?

| Approach          | CGO?   | External deps         | Cross-platform         | Reliable |
| ----------------- | ------ | --------------------- | ---------------------- | -------- |
| Fyne (native)     | ✅ Yes | MinGW + OpenGL + X11  | ❌ Linux needs dev libs | ❌       |
| **Web UI (this)** | ❌ No  | Just a browser        | ✅ Everywhere           | ✅       |

The web GUI uses:

- **Pure Go standard library** — no CGO, no compiler toolchain needed.
- **Embedded HTML/CSS/JS** — all UI assets are bundled into the binary.
- **Server-Sent Events (SSE)** — live progress streaming to the browser.
- **Vazirmatn font** — full Persian + Latin glyph coverage.
- **Dark modern theme** with an indigo accent.
- **Bilingual** with a one-click language toggle (FA ⇄ EN, RTL ⇄ LTR).

### GUI features

- **Sidebar navigation** (VS Code-style): Settings, Controls, Progress,
  Advanced, Logs, About.
- **Live progress**: page/asset counters, byte count, elapsed time, current
  URL, last saved file, and a progress bar.
- **Real-time log viewer** with filter and copy.
- **Open output** button to launch the file manager at the mirror dir.
- **Open in browser** button to preview the mirrored `index.html`.

## Mirror layout

`webclone` preserves the URL hierarchy exactly:

| Source URL                            | On-disk path                        |
| ------------------------------------- | ----------------------------------- |
| `https://example.com/`                | `example.com/index.html`            |
| `https://example.com/foo`             | `example.com/foo/index.html`        |
| `https://example.com/foo/`            | `example.com/foo/index.html`        |
| `https://example.com/foo/bar`         | `example.com/foo/bar/index.html`    |
| `https://example.com/foo/bar.html`    | `example.com/foo/bar.html`          |
| `https://example.com/style.css`       | `example.com/style.css`             |
| `https://example.com/foo?x=1`         | `example.com/foo/index_<hash>.html` |
| `https://example.com/style.css?v=123` | `example.com/style_<hash>.css`      |

URLs with a query string get a short MD5 hash of the query appended to the
filename (before the extension), so distinct query-keyed pages get distinct
files.

## Link rewriting

After downloading a page, `webclone` rewrites every link inside it to point at
the local mirror copy. The rewriter handles:

- `<a href>` and `<area href>` — page links
- `<iframe src>` — iframe pages
- `<link rel="stylesheet|icon|manifest|...">` — CSS / icons / manifests
- `<script src>` — JavaScript
- `<img src>` and `<img srcset>`
- `<source src>` and `<source srcset>`
- `<video src>`, `<video poster>`, `<audio src>`
- `<embed src>`, `<object data>`
- Inline `style="background: url(...)"`
- `<style>` blocks (rewrites `url()` and `@import`)
- `<meta http-equiv="refresh" content="0; url=...">`

URLs that are not being mirrored (external links when `--same-domain` is on, or
`mailto:` / `javascript:` / `data:` schemes) are left untouched.

## Output structure

```
./example.com/
├── index.html              # https://example.com/
├── about/
│   └── index.html          # https://example.com/about
├── blog/
│   ├── index.html          # https://example.com/blog
│   └── post-1/
│       └── index.html      # https://example.com/blog/post-1
├── css/
│   └── main.css            # https://example.com/css/main.css
├── js/
│   └── app.js
├── imgs/
│   └── logo.png
└── manifest.json           # List of every URL crawled
```

## Building from source

Requires **Go 1.22+**.

```bash
# CLI
go build -o webclone ./cmd/webclone

# GUI
go build -o webclone-gui ./cmd/webclone-gui
```

Or use the provided `Makefile`:

```bash
make build     # build the CLI
make gui       # build the GUI
make test      # run tests with the race detector
make dist      # cross-compile for Linux, macOS, and Windows
```

Cross-compile manually:

```bash
GOOS=linux   GOARCH=amd64 go build -o webclone-linux-amd64   ./cmd/webclone
GOOS=darwin  GOARCH=arm64 go build -o webclone-darwin-arm64  ./cmd/webclone
GOOS=windows GOARCH=amd64 go build -o webclone.exe           ./cmd/webclone
```

## Project layout

```
.
├── cmd/
│   ├── webclone/         # CLI entry point
│   └── webclone-gui/     # GUI entry point
├── pkg/
│   ├── cli/              # cobra command + flag wiring
│   ├── crawler/          # recursive crawl engine, worker pool, events
│   ├── download/         # HTTP fetch client
│   ├── parse/            # HTML + CSS parsing and link rewriting
│   ├── urlx/             # URL canonicalization and path mapping
│   └── webui/            # browser-based GUI server + embedded assets
├── Makefile
├── go.mod
└── README.md
```

## Differences from `goclone`

| Aspect                   | `goclone`                                                   | `webclone`                                                                     |
| ------------------------ | ----------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Scope                    | Single page                                                 | Entire site (recursive)                                                        |
| Mirror layout            | `index.html` + flat `css/`, `js/`, `imgs/`                  | URL hierarchy preserved exactly                                                |
| Link rewriting           | Only `<script src>`, `<link>`, `<img src>` (top-level page) | All link types, including `srcset`, `<style>`, inline `style`, `<iframe>`, etc. |
| CSS rewriting            | No                                                          | Yes (`url()`, `@import`)                                                        |
| Concurrency              | colly async                                                 | Worker pool with bounded concurrency                                           |
| External link following  | No                                                          | Configurable (off by default via `--same-domain`)                              |
| Domain policy            | N/A                                                         | `--same-domain`, `--allowed-hosts`                                             |
| Manifest                 | No                                                          | Yes                                                                            |
| GUI                      | No                                                          | Browser-based, bilingual (FA/EN)                                               |

## License

MIT — see [LICENSE](LICENSE).

## Acknowledgements

Inspired by [`goclone-dev/goclone`](https://github.com/goclone-dev/goclone) and
`wget --mirror`.
