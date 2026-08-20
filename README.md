# webclone
<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="320" viewBox="0 0 1200 320">
  <!-- Background -->
  <defs>
    <linearGradient id="bgGradient" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#0d1117" />
      <stop offset="100%" stop-color="#161b22" />
    </linearGradient>
    <linearGradient id="textGradient" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stop-color="#58a6ff" />
      <stop offset="50%" stop-color="#79c0ff" />
      <stop offset="100%" stop-color="#a371f7" />
    </linearGradient>
    <filter id="glow">
      <feGaussianBlur stdDeviation="5" result="coloredBlur"/>
      <feMerge>
        <feMergeNode in="coloredBlur"/>
        <feMergeNode in="SourceGraphic"/>
      </feMerge>
    </filter>
  </defs>

  <!-- Main Background Rect -->
  <rect width="1200" height="320" rx="20" fill="url(#bgGradient)" />
  
  <!-- Decorative Background Shapes -->
  <circle cx="1050" cy="60" r="120" fill="#1f6feb" opacity="0.15" />
  <circle cx="1150" cy="260" r="180" fill="#a371f7" opacity="0.1" />
  <path d="M 0 250 Q 600 320 1200 200" stroke="#30363d" stroke-width="2" fill="none" opacity="0.5"/>

  <!-- Left Side: Text Content -->
  <text x="100" y="140" font-family="Segoe UI, Helvetica, sans-serif" font-size="72" font-weight="bold" fill="url(#textGradient)" filter="url(#glow)">
    Web Clone
  </text>
  <text x="100" y="185" font-family="Segoe UI, Helvetica, sans-serif" font-size="24" fill="#c9d1d9" opacity="0.9">
    A modern web cloning project by A-talebifard
  </text>
  <text x="100" y="225" font-family="monospace" font-size="18" fill="#8b949e">
    &lt;html&gt; &lt;css/&gt; &lt;javascript/&gt; ✨
  </text>

  <!-- Right Side: Browser Mockup Graphic -->
  <g transform="translate(750, 70)">
    <!-- Browser Window -->
    <rect x="0" y="0" width="350" height="180" rx="10" fill="#161b22" stroke="#30363d" stroke-width="2" />
    <rect x="0" y="0" width="350" height="30" rx="10" fill="#21262d" />
    <!-- Browser Dots -->
    <circle cx="20" cy="15" r="6" fill="#ff5f56" />
    <circle cx="40" cy="15" r="6" fill="#ffbd2e" />
    <circle cx="60" cy="15" r="6" fill="#27c93f" />
    <!-- Browser URL Bar -->
    <rect x="90" y="8" width="240" height="14" rx="7" fill="#0d1117" />
    
    <!-- Code Lines Mockup inside Browser -->
    <rect x="20" y="50" width="200" height="8" rx="4" fill="#58a6ff" opacity="0.8" />
    <rect x="20" y="70" width="280" height="6" rx="3" fill="#c9d1d9" opacity="0.5" />
    <rect x="20" y="85" width="250" height="6" rx="3" fill="#c9d1d9" opacity="0.5" />
    <rect x="40" y="100" width="220" height="6" rx="3" fill="#a371f7" opacity="0.6" />
    <rect x="20" y="115" width="260" height="6" rx="3" fill="#c9d1d9" opacity="0.5" />
    <rect x="20" y="135" width="160" height="8" rx="4" fill="#3fb950" opacity="0.8" />
  </g>
</svg>
> ابزار سریع و کامل برای آینه‌سازی (mirror) کل یک وب‌سایت روی دیسک محلی — نوشته‌شده با Go، همراه با نسخه خط‌فرمان (CLI) و رابط گرافیکی مبتنی بر مرورگر.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](#ساخت-از-روی-سورس)

**زبان:** [فارسی](#-راهنمای-فارسی) · [English](#-english-guide)

---

# 🇮🇷 راهنمای فارسی

`webclone` کل یک وب‌سایت را روی دیسک محلی شما آینه‌سازی می‌کند. برخلاف
ابزارهای تک‌صفحه‌ای، این ابزار لینک‌های داخلی را به‌صورت بازگشتی دنبال می‌کند و
هر صفحه و هر دارایی (HTML، CSS، JS، تصاویر، فونت‌ها، ویدئو/صدا و ...) را ذخیره
می‌کند و ساختار آدرس‌های سایت را زیر پوشه خروجی بازتولید می‌کند تا نسخه آینه‌شده
به‌طور **کاملاً آفلاین** قابل مرور باشد.

این ابزار در دو قالب ارائه می‌شود:

- **`webclone`** — ابزار خط‌فرمان برای اسکریپت‌نویسی و خودکارسازی.
- **`webclone-gui`** — رابط گرافیکی دسکتاپ مبتنی بر مرورگر (Go خالص، بدون CGO)
  با رابط دوزبانه (فارسی / انگلیسی)، تم تیره و نمایش زنده پیشرفت.

این پروژه بازنویسی از پایه و با الهام از
[`goclone`](https://github.com/goclone-dev/goclone) است، با این تفاوت که به‌جای
کلون تک‌صفحه‌ای، کل سایت را به‌صورت بازگشتی می‌خزد.

## امکانات

- **خزش بازگشتی** — دنبال‌کردن `<a href>`، `<iframe src>` و (به‌صورت اختیاری)
  لینک‌های خارجی.
- **چیدمان آینه‌ای** — آدرس `https://example.com/foo/bar` در مسیر
  `example.com/foo/bar/index.html` ذخیره می‌شود، پس ساختار روی دیسک دقیقاً
  سلسله‌مراتب آدرس‌ها را بازتاب می‌دهد.
- **بازنویسی لینک‌ها** — هر آدرس مطلق یا ریشه‌نسبی داخل HTML/CSS دانلودشده به یک
  مسیر نسبی که به نسخه محلی اشاره می‌کند بازنویسی می‌شود، تا آینه بدون هیچ
  بازنویسی سمت سرور، آفلاین کار کند.
- **پوشش کامل دارایی‌ها** — CSS (شامل `url()` و `@import`)، JS، تصاویر
  (`src` و `srcset`)، فونت‌ها، ویدئو/صدا، `style="..."` درون‌خطی، بلوک‌های
  `<style>`، `<source>`، `<embed>`، `<object>`، `<iframe>` و
  `<meta http-equiv="refresh">`.
- **همزمانی** — استخر کارگرِ محدود (پیش‌فرض ۵، قابل تنظیم).
- **سقف‌های ایمنی** — `--max-urls` و `--max-depth` جلوی خزش بی‌پایان را می‌گیرند.
- **محدودیت دامنه** — `--same-domain` (به‌صورت پیش‌فرض روشن)،
  `--allow-subdomains` و `--allowed-hosts` کنترل دقیق روی میزبان‌های مجاز می‌دهند.
- **پشتیبانی از کوکی** — تنظیم کوکی با `--cookie` برای سایت‌هایی که نیاز به
  نشست (session) دارند.
- **پروکسی + UA سفارشی + TLS ناامن** — پیکربندی کامل کلاینت HTTP.
- **مانیفست** — `--manifest` (پیش‌فرض روشن) فایل `manifest.json` شامل همه
  آدرس‌های خزیده‌شده را می‌نویسد.
- **سرور فایل داخلی** — `--serve` یک سرور HTTP ایستا اجرا می‌کند تا بلافاصله
  آینه را پیش‌نمایش کنید.
- **رابط گرافیکی مرورگری** — چندسکویی، بدون وابستگی خارجی، نمایش زنده پیشرفت با
  Server-Sent Events، دوزبانه فارسی/انگلیسی با تغییر جهت RTL/LTR.

## نصب

از روی سورس:

```bash
git clone https://github.com/a-talebifard/webclone.git
cd webclone
go build -o webclone ./cmd/webclone
./webclone --help
```

یا با نصب‌بودن Go:

```bash
go install github.com/a-talebifard/webclone/cmd/webclone@latest
```

## استفاده از خط‌فرمان

```bash
webclone [flags] <url> [<url2> ...]
```

### مثال‌های رایج

```bash
# آینه‌سازی یک سایت (پیش‌فرض فقط همان دامنه)، همه صفحات داخلی:
webclone https://example.com

# آینه‌سازی در یک پوشه خروجی مشخص:
webclone -o ./mirror https://example.com

# محدودکردن به ۳ پرش لینک از نقطه شروع (سریع‌تر و کوچک‌تر):
webclone --max-depth 3 https://example.com

# دنبال‌کردن لینک‌های خارجی هم (خاموش‌کردن محدودیت هم‌دامنه):
webclone --same-domain=false https://example.com

# کارگر سفارشی + UA سفارشی + پروکسی:
webclone -w 10 -u "Mozilla/5.0 ..." --proxy http://localhost:8080 https://example.com

# فهرست میزبان‌های مجاز صریح (بر --same-domain اولویت دارد):
webclone --allowed-hosts example.com,cdn.example.com https://example.com

# آینه‌سازی و سرو فوری روی http://localhost:8080:
webclone -s https://example.com

# تنظیم کوکی از قبل (برای سایت‌هایی که نشست می‌خواهند):
webclone -C "session=abc123; token=xyz" https://example.com
```

### همه پرچم‌ها (flags)

| پرچم                 | پیش‌فرض   | توضیح                                                                 |
| -------------------- | --------- | --------------------------------------------------------------------- |
| `-o, --output`       | `.`       | پوشه خروجی برای درخت آینه                                              |
| `-w, --workers`      | `5`       | تعداد کارگرهای همزمان دریافت                                           |
| `--max-urls`         | `10000`   | سقف کل آدرس‌های دریافتی (۰ = نامحدود)                                  |
| `--max-depth`        | `0`       | حداکثر پرش لینک از نقطه شروع (۰ = نامحدود)                             |
| `--same-domain`      | `true`    | فقط آدرس‌های روی دامنه ثبت‌شده نقطه شروع را بخز                        |
| `--allow-subdomains` | `true`    | وقتی `--same-domain` روشن است، زیردامنه‌های میزبان را هم دنبال کن       |
| `--allowed-hosts`    | (خالی)    | فهرست صریح میزبان‌ها با کاما (بر `--same-domain` اولویت دارد)          |
| `--skip-assets`      | `false`   | فقط صفحات HTML را دانلود کن، از CSS/JS/تصاویر بگذر                     |
| `--asset-ext`        | (خالی)    | فهرست پسوندهای دارایی برای دانلود با کاما (خالی = همه)                 |
| `--timeout`          | `60s`     | مهلت هر درخواست                                                       |
| `-u, --user-agent`   | (پیش‌فرض) | رشته User-Agent سفارشی                                                |
| `-p, --proxy`        | (خالی)    | آدرس پروکسی (`http://`، `https://`، `socks5://`)                      |
| `--insecure-tls`     | `false`   | نادیده‌گرفتن اعتبارسنجی گواهی TLS                                     |
| `-v, --verbose`      | `false`   | خروجی گزارش پرجزئیات به‌ازای هر آدرس                                   |
| `-q, --quiet`        | `false`   | حذف همه خروجی‌های غیرخطا                                              |
| `-s, --serve`        | `false`   | پس از آینه‌سازی، پوشه خروجی را روی یک سرور HTTP محلی سرو کن            |
| `-P, --serve-port`   | `8080`    | پورت برای `--serve`                                                  |
| `--manifest`         | `true`    | نوشتن `manifest.json` با همه آدرس‌های خزیده‌شده                        |
| `-C, --cookie`       | (خالی)    | تنظیم کوکی از قبل (`name=value; name2=value2`)                        |

## رابط گرافیکی (GUI)

`webclone-gui` یک رابط گرافیکی دسکتاپ **مبتنی بر مرورگر** است. یک سرور HTTP
محلی اجرا می‌کند و مرورگر پیش‌فرض شما را باز می‌کند — بدون CGO، بدون MinGW و
بدون هیچ وابستگی خارجی، فقط Go خالص.

### ساخت و اجرا

```bash
go build -o webclone-gui ./cmd/webclone-gui
./webclone-gui
```

در ویندوز:

```powershell
go build -o webclone-gui.exe .\cmd\webclone-gui
.\webclone-gui.exe
```

مرورگر شما به‌صورت خودکار روی `http://127.0.0.1:8080/` باز می‌شود. برای پورت
سفارشی:

```bash
./webclone-gui 9090   # به‌جای 8080 از پورت 9090 استفاده کن
```

### امکانات رابط گرافیکی

- **ناوبری کناری** (به سبک VS Code): تنظیمات، کنترل‌ها، پیشرفت، پیشرفته،
  گزارش‌ها، درباره.
- **پیشرفت زنده**: شمارنده صفحه/دارایی، تعداد بایت، زمان سپری‌شده، آدرس فعلی،
  آخرین فایل ذخیره‌شده و نوار پیشرفت.
- **نمایشگر گزارش بلادرنگ** با فیلتر و کپی.
- **دوزبانه** با تغییر زبان تک‌کلیکی (FA ⇄ EN، RTL ⇄ LTR).
- دکمه **باز کردن خروجی** برای اجرای مدیر فایل در پوشه آینه.
- دکمه **باز کردن در مرورگر** برای پیش‌نمایش `index.html` آینه‌شده.

## چیدمان آینه

`webclone` سلسله‌مراتب آدرس‌ها را دقیقاً حفظ می‌کند:

| آدرس منبع                             | مسیر روی دیسک                        |
| ------------------------------------- | ----------------------------------- |
| `https://example.com/`                | `example.com/index.html`            |
| `https://example.com/foo`             | `example.com/foo/index.html`        |
| `https://example.com/foo/bar.html`    | `example.com/foo/bar.html`          |
| `https://example.com/style.css`       | `example.com/style.css`             |
| `https://example.com/foo?x=1`         | `example.com/foo/index_<hash>.html` |

آدرس‌های دارای رشته پرس‌وجو (query) یک هش کوتاه MD5 از query را در انتهای نام
فایل (پیش از پسوند) می‌گیرند، تا صفحات متمایز با query متفاوت، فایل‌های جدا
بگیرند.

## ساخت از روی سورس

نیازمند **Go 1.22 یا بالاتر**.

```bash
# نسخه خط‌فرمان
go build -o webclone ./cmd/webclone

# نسخه گرافیکی
go build -o webclone-gui ./cmd/webclone-gui
```

یا با `Makefile` موجود:

```bash
make build     # ساخت CLI
make gui       # ساخت GUI
make test      # اجرای تست‌ها با آشکارساز race
make dist      # کامپایل متقاطع برای Linux، macOS و Windows
```

## مجوز

MIT — به [LICENSE](LICENSE) مراجعه کنید.

---

# 🇬🇧 English Guide

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

### GUI features

- **Sidebar navigation** (VS Code-style): Settings, Controls, Progress,
  Advanced, Logs, About.
- **Live progress**: page/asset counters, byte count, elapsed time, current
  URL, last saved file, and a progress bar.
- **Real-time log viewer** with filter and copy.
- **Bilingual** with a one-click language toggle (FA ⇄ EN, RTL ⇄ LTR).
- **Open output** button to launch the file manager at the mirror dir.
- **Open in browser** button to preview the mirrored `index.html`.

## Mirror layout

`webclone` preserves the URL hierarchy exactly:

| Source URL                            | On-disk path                        |
| ------------------------------------- | ----------------------------------- |
| `https://example.com/`                | `example.com/index.html`            |
| `https://example.com/foo`             | `example.com/foo/index.html`        |
| `https://example.com/foo/bar.html`    | `example.com/foo/bar.html`          |
| `https://example.com/style.css`       | `example.com/style.css`             |
| `https://example.com/foo?x=1`         | `example.com/foo/index_<hash>.html` |

URLs with a query string get a short MD5 hash of the query appended to the
filename (before the extension), so distinct query-keyed pages get distinct
files.

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

## License

MIT — see [LICENSE](LICENSE).

## Acknowledgements

Inspired by [`goclone-dev/goclone`](https://github.com/goclone-dev/goclone) and
`wget --mirror`.
