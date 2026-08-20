// Package urlx provides URL utilities for the webclone crawler.
//
// It is responsible for:
//   - Canonicalizing URLs (resolving relative links, normalizing)
//   - Filtering out non-crawlable schemes (mailto:, javascript:, data:, blob:, ...)
//   - Mapping a URL to a deterministic local filesystem path (the "mirror" layout)
//   - Computing relative paths between two local mirror paths so that
//     downloaded HTML can keep links working offline
package urlx

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// IsCrawlable returns true if the URL has a scheme we can actually fetch
// and follow. It deliberately accepts both http and https. Schemes such as
// mailto:, tel:, javascript:, data:, blob:, ftp:, file: are rejected so the
// crawler does not enqueue them as pages.
func IsCrawlable(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	// Strip leading/trailing whitespace
	rawURL = strings.TrimSpace(rawURL)
	// Reject obvious non-link protocols even before parsing
	lower := strings.ToLower(rawURL)
	for _, bad := range []string{"mailto:", "tel:", "javascript:", "data:", "blob:", "ftp:", "file:", "ws:", "wss:"} {
		if strings.HasPrefix(lower, bad) {
			return false
		}
	}
	// Anchor-only links (#...) are never crawlable as separate pages
	if strings.HasPrefix(rawURL, "#") {
		return false
	}
	return true
}

// IsHTTPScheme returns true if the parsed URL uses http or https.
func IsHTTPScheme(u *url.URL) bool {
	s := strings.ToLower(u.Scheme)
	return s == "http" || s == "https"
}

// Canonicalize resolves a possibly-relative URL against a base URL and
// returns a normalized absolute URL string with the fragment removed.
//
// It returns an empty string if the input cannot be parsed or if the
// resolved URL does not have an http/https scheme.
func Canonicalize(rawURL, baseURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("empty url")
	}

	// Parse base first; if it fails, we cannot resolve relative URLs.
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Strip the fragment from the URL before resolving, so that
	// "page.html#section" is treated as just "page.html".
	ref, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	ref.Fragment = ""

	resolved := base.ResolveReference(ref)
	if !IsHTTPScheme(resolved) {
		return "", errors.New("non-http(s) scheme")
	}

	// Remove default ports (e.g. :80 on http, :443 on https)
	resolved.Host = stripDefaultPort(resolved.Host, resolved.Scheme)
	// Normalize empty path to "/"
	if resolved.Path == "" {
		resolved.Path = "/"
	}
	return resolved.String(), nil
}

// stripDefaultPort removes the default port for the given scheme.
// e.g. http://example.com:80/ -> example.com
func stripDefaultPort(host, scheme string) string {
	scheme = strings.ToLower(scheme)
	if strings.HasSuffix(host, ":80") && scheme == "http" {
		return strings.TrimSuffix(host, ":80")
	}
	if strings.HasSuffix(host, ":443") && scheme == "https" {
		return strings.TrimSuffix(host, ":443")
	}
	return host
}

// MirrorPath converts an absolute URL into a deterministic local filesystem
// path that mirrors the site's URL structure.
//
// Examples:
//
//	https://example.com/                       -> example.com/index.html
//	https://example.com                         -> example.com/index.html
//	https://example.com/foo                     -> example.com/foo/index.html
//	https://example.com/foo/                    -> example.com/foo/index.html
//	https://example.com/foo/bar                 -> example.com/foo/bar/index.html
//	https://example.com/foo/bar.html            -> example.com/foo/bar.html
//	https://example.com/style.css               -> example.com/style.css
//	https://example.com/foo?x=1                 -> example.com/foo/index_<hash>.html
//	https://example.com/style.css?v=123         -> example.com/style_<hash>.css
//
// The function is idempotent and deterministic: the same input always
// produces the same output.
func MirrorPath(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if !IsHTTPScheme(u) {
		return "", errors.New("non-http(s) scheme")
	}

	host := u.Hostname()
	if host == "" {
		host = "local"
	}
	// Lowercase the host for case-insensitive consistency. Path stays case
	// sensitive because URLs are case-sensitive after the host.
	host = strings.ToLower(host)

	urlPath := u.Path
	if urlPath == "" {
		urlPath = "/"
	}
	urlPath = strings.TrimPrefix(urlPath, "/")
	if urlPath == "" {
		// Root URL: example.com/index.html
		return filepath.Join(host, "index.html"), nil
	}

	segments := strings.Split(urlPath, "/")
	// Remove trailing empty segment from paths like "/foo/"
	if len(segments) > 0 && segments[len(segments)-1] == "" {
		segments = segments[:len(segments)-1]
	}
	if len(segments) == 0 {
		return filepath.Join(host, "index.html"), nil
	}

	lastIdx := len(segments) - 1
	last := segments[lastIdx]
	ext := path.Ext(last)

	// Determine the filename portion.
	var filename string
	if ext == "" {
		// No extension: treat the last segment as a directory and put index.html inside it.
		// e.g. /foo/bar -> foo/bar/index.html
		segments = append(segments, "index.html")
		filename = "index.html"
	} else {
		filename = last
	}

	// If the URL has a query string, disambiguate by appending a short hash
	// of the query to the filename (before the extension). This lets us keep
	// distinct pages that share the same path but differ by query.
	if u.RawQuery != "" {
		h := md5.Sum([]byte(u.RawQuery))
		qHash := hex.EncodeToString(h[:])[:8]
		if ext == "" {
			// segments already has the appended "index.html"
			segments[lastIdx+1] = "index_" + qHash + ".html"
		} else {
			base := strings.TrimSuffix(filename, ext)
			segments[lastIdx] = base + "_" + qHash + ext
		}
	}

	// Replace forward slashes in segments that came from the URL path with
	// the OS-specific separator when joining. We use filepath.Join which
	// also cleans ../.. attempts.
	// NOTE: on Windows the join would use backslashes; for a portable mirror
	// we always want forward slashes. We post-process to normalize.
	joined := filepath.Join(append([]string{host}, segments...)...)
	return filepath.ToSlash(joined), nil
}

// MirrorDir returns the directory portion of MirrorPath(url). This is the
// directory the mirrored file lives in; useful when computing relative
// links between two mirrored pages.
func MirrorDir(rawURL string) (string, error) {
	p, err := MirrorPath(rawURL)
	if err != nil {
		return "", err
	}
	// Use forward-slash semantics for portability.
	dir := path.Dir(p)
	if dir == "." {
		dir = ""
	}
	return dir, nil
}

// RelativePath computes a relative path from a "from" directory to a "to"
// filesystem path, both expressed with forward slashes. The result is also
// forward-slashed and is suitable for embedding in HTML href attributes.
//
// Examples:
//
//	RelativePath("example.com/foo", "example.com/about.html") -> "../../about.html"
//	RelativePath("example.com", "example.com/about.html")      -> "about.html"
//	RelativePath("example.com/foo", "example.com/foo/bar.html") -> "bar.html"
//	RelativePath("example.com/foo", "other.com/page/index.html") -> "../../other.com/page/index.html"
func RelativePath(fromDir, toPath string) string {
	fromParts := splitNonEmpty(fromDir)
	toParts := splitNonEmpty(toPath)
	if len(fromParts) == 0 && len(toParts) == 0 {
		return ""
	}

	// Find length of common prefix.
	common := 0
	for common < len(fromParts) && common < len(toParts) && fromParts[common] == toParts[common] {
		common++
	}

	ups := len(fromParts) - common
	downs := toParts[common:]

	parts := make([]string, 0, ups+len(downs))
	for i := 0; i < ups; i++ {
		parts = append(parts, "..")
	}
	parts = append(parts, downs...)
	if len(parts) == 0 {
		// Same file: just point at itself.
		return "."
	}
	return strings.Join(parts, "/")
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// IsHTMLAsset heuristically returns true if the URL likely points to an HTML
// document. We use this to decide whether to crawl the URL for further links
// (HTML yes) or just download it as a static asset (HTML no).
func IsHTMLAsset(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	p := strings.ToLower(u.Path)
	// No extension or known HTML extensions
	ext := path.Ext(p)
	if ext == "" {
		// /foo, /foo/ - assume HTML page (server renders HTML by default)
		return true
	}
	switch ext {
	case ".html", ".htm", ".xhtml", ".shtml", ".asp", ".aspx", ".jsp", ".php", ".cgi":
		return true
	}
	return false
}

// IsPageLike returns true if the URL should be enqueued for crawling
// (rather than just downloaded as a binary asset). For our purposes this
// means IsHTMLAsset(u) is true AND the URL is crawlable.
func IsPageLike(rawURL string) bool {
	if !IsCrawlable(rawURL) {
		return false
	}
	return IsHTMLAsset(rawURL)
}
