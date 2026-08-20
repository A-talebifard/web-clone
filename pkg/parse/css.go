package parse

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/a-talebifard/webclone/pkg/urlx"
)

// CSS url() patterns. Go's RE2-based regexp does not support backreferences,
// so we split the pattern into three alternatives for the three quoting
// styles. Each pattern captures the inner URL in group 1.
//
//	url("foo.png")    -> double-quoted  (cssURLDQ)
//	url('foo.png')    -> single-quoted  (cssURLSQ)
//	url(foo.png)      -> bare            (cssURLBare)
var (
	cssURLDQ   = regexp.MustCompile(`url\(\s*"([^"]+)"\s*\)`)
	cssURLSQ   = regexp.MustCompile(`url\(\s*'([^']+)'\s*\)`)
	cssURLBare = regexp.MustCompile(`url\(\s*([^'"\s)]+)\s*\)`)

	// @import "foo.css";  or  @import 'foo.css';  (the url(...) form is
	// already handled by the url() regexes above)
	cssImportDQ = regexp.MustCompile(`@import\s+"([^"]+)"`)
	cssImportSQ = regexp.MustCompile(`@import\s+'([^']+)'`)
)

// RewriteCSS rewrites every url() and @import reference inside a CSS body so
// that it points to the local mirror copy relative to the page/dir the CSS
// is associated with.
//
// pageURL is the absolute URL of the document the CSS lives in (a page URL
// for inline <style> or a .css file URL for external stylesheets). pageDir
// is the local mirror directory of pageURL (forward-slashed). shouldMirror
// decides whether each discovered URL should be downloaded/rewritten.
//
// Returns the rewritten CSS plus the list of asset URLs discovered.
func RewriteCSS(body, pageURL, pageDir string, shouldMirror func(string) bool) (string, []string) {
	var urls []string

	rewriteRef := func(rawRef string) string {
		rawRef = strings.TrimSpace(rawRef)
		if rawRef == "" || strings.HasPrefix(rawRef, "data:") || strings.HasPrefix(rawRef, "#") {
			return rawRef
		}
		canonical, err := urlx.Canonicalize(rawRef, pageURL)
		if err != nil {
			return rawRef
		}
		if !urlx.IsCrawlable(canonical) || !shouldMirror(canonical) {
			return canonical
		}
		targetPath, err := urlx.MirrorPath(canonical)
		if err != nil {
			return rawRef
		}
		urls = append(urls, canonical)
		return urlx.RelativePath(pageDir, targetPath)
	}

	// 1. Rewrite double-quoted url("...") references.
	body = cssURLDQ.ReplaceAllStringFunc(body, func(m string) string {
		sub := cssURLDQ.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		newURL := rewriteRef(sub[1])
		return `url("` + newURL + `")`
	})

	// 2. Rewrite single-quoted url('...') references.
	body = cssURLSQ.ReplaceAllStringFunc(body, func(m string) string {
		sub := cssURLSQ.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		newURL := rewriteRef(sub[1])
		return `url('` + newURL + `')`
	})

	// 3. Rewrite bare url(foo) references. Run AFTER the quoted forms so we
	// don't double-match; the quoted regexes above have already replaced
	// every `url("...")` and `url('...')` with the same syntax, so the bare
	// pattern will not see them.
	body = cssURLBare.ReplaceAllStringFunc(body, func(m string) string {
		sub := cssURLBare.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		newURL := rewriteRef(sub[1])
		if needsCSSQuote(newURL) {
			return `url("` + newURL + `")`
		}
		return `url(` + newURL + `)`
	})

	// 4. Rewrite @import "..." and @import '...'.
	body = cssImportDQ.ReplaceAllStringFunc(body, func(m string) string {
		sub := cssImportDQ.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return `@import "` + rewriteRef(sub[1]) + `"`
	})
	body = cssImportSQ.ReplaceAllStringFunc(body, func(m string) string {
		sub := cssImportSQ.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return `@import '` + rewriteRef(sub[1]) + `'`
	})

	return body, urls
}

// needsCSSQuote returns true when the URL contains characters that must be
// quoted inside an unquoted url() reference (per CSS spec).
func needsCSSQuote(s string) bool {
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '(', ')', ',', '"', '\'':
			return true
		}
	}
	return false
}

// ShortHash returns a short hex MD5 of the input, useful for unique
// filenames when needed (e.g. embedded data: URLs the caller chooses to
// save as files).
func ShortHash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])[:8]
}

// EscapeCSSString escapes a string for safe inclusion inside CSS quotes.
func EscapeCSSString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// ParseCSSURLs returns every URL referenced inside url() and @import
// statements of the given CSS body, canonicalized against pageURL.
// It does NOT rewrite the CSS - it only extracts. Useful when we want to
// fetch the assets without rewriting (e.g. when CSS came from a CDN we
// are not mirroring).
func ParseCSSURLs(body, pageURL string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "#") {
			return
		}
		canonical, err := urlx.Canonicalize(raw, pageURL)
		if err != nil {
			return
		}
		if !urlx.IsCrawlable(canonical) {
			return
		}
		if _, ok := seen[canonical]; ok {
			return
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}

	for _, re := range []*regexp.Regexp{cssURLDQ, cssURLSQ, cssURLBare} {
		for _, m := range re.FindAllStringSubmatch(body, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	for _, re := range []*regexp.Regexp{cssImportDQ, cssImportSQ} {
		for _, m := range re.FindAllStringSubmatch(body, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return out
}
