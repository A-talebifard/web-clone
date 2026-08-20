// Package parse extracts links from HTML/CSS and rewrites them so the
// downloaded page can be browsed offline.
//
// The rewriter converts every absolute or site-relative URL into a relative
// path that points at the mirrored copy on disk. For URLs we did not
// download (e.g. external pages when --same-domain-only is set, or non-http
// links), the original attribute value is left untouched.
package parse

import (
        "bytes"
        "fmt"
        "net/url"
        "regexp"
        "strings"

        "github.com/PuerkitoBio/goquery"
        "golang.org/x/net/html"

        "github.com/a-talebifard/webclone/pkg/urlx"
)

// HTMLRewriteResult is the rewritten HTML plus the set of URLs that the
// caller should consider downloading/crawling next.
type HTMLRewriteResult struct {
        HTML      []byte
        PageLinks []string // hrefs from <a> tags, plus <iframe src>
        Assets    []string // CSS/JS/img/font/media URLs that should be fetched as-is
}

// RewriteHTML parses the HTML body found at pageURL, rewrites all internal
// links to be relative to the local mirror layout, and returns the rewritten
// HTML plus the list of discovered URLs (links + assets).
//
// shouldMirror(url) is called for every URL we discover; only when it
// returns true do we rewrite the attribute to point at the local mirror.
// Otherwise the original URL is kept untouched. This lets the caller decide
// policy (e.g. "only rewrite URLs whose host is on the crawl allowlist")
// without baking policy into this package.
func RewriteHTML(body []byte, pageURL string, shouldMirror func(string) bool) (*HTMLRewriteResult, error) {
        doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
        if err != nil {
                return nil, fmt.Errorf("parse html %s: %w", pageURL, err)
        }

        res := &HTMLRewriteResult{}

        pageDir, _ := urlx.MirrorDir(pageURL)

        // Helper: given a target URL, decide if we should rewrite it to a local
        // mirror path. Returns the local relative path or empty string if not.
        localRel := func(rawTarget string) (string, string, error) {
                canonical, err := urlx.Canonicalize(rawTarget, pageURL)
                if err != nil {
                        return "", "", err
                }
                if !shouldMirror(canonical) {
                        return canonical, "", nil
                }
                targetPath, err := urlx.MirrorPath(canonical)
                if err != nil {
                        return canonical, "", err
                }
                rel := urlx.RelativePath(pageDir, targetPath)
                return canonical, rel, nil
        }

        // 1. <a href="..."> and <area href="..."> -> page links
        // Also <iframe src="..."> (page-like).
        handlePageLink := func(sel *goquery.Selection, attr string) {
                val, exists := sel.Attr(attr)
                if !exists || val == "" {
                        return
                }
                canonical, rel, err := localRel(val)
                if err == nil && rel != "" {
                        sel.SetAttr(attr, rel)
                }
                if canonical != "" && urlx.IsCrawlable(canonical) {
                        res.PageLinks = append(res.PageLinks, canonical)
                }
        }
        doc.Find("a[href], area[href]").Each(func(_ int, s *goquery.Selection) {
                handlePageLink(s, "href")
        })
        doc.Find("iframe[src]").Each(func(_ int, s *goquery.Selection) {
                handlePageLink(s, "src")
        })

        // 2. Assets: <link rel="stylesheet">, <link rel="icon">, <link rel="manifest">, etc.
        // Treat every <link href> as an asset except preconnect / dns-prefetch
        // which only point at origins (not files).
        doc.Find("link[href]").Each(func(_ int, s *goquery.Selection) {
                rel, _ := s.Attr("rel")
                rel = strings.ToLower(strings.TrimSpace(rel))
                if rel == "preconnect" || rel == "dns-prefetch" {
                        return
                }
                val, exists := s.Attr("href")
                if !exists || val == "" {
                        return
                }
                canonical, relPath, err := localRel(val)
                if err == nil && relPath != "" {
                        s.SetAttr("href", relPath)
                }
                if canonical != "" && urlx.IsCrawlable(canonical) {
                        res.Assets = append(res.Assets, canonical)
                }
        })

        // 3. <script src>
        doc.Find("script[src]").Each(func(_ int, s *goquery.Selection) {
                val, exists := s.Attr("src")
                if !exists || val == "" {
                        return
                }
                canonical, relPath, err := localRel(val)
                if err == nil && relPath != "" {
                        s.SetAttr("src", relPath)
                }
                if canonical != "" && urlx.IsCrawlable(canonical) {
                        res.Assets = append(res.Assets, canonical)
                }
        })

        // 4. <img src> and <img srcset>
        doc.Find("img").Each(func(_ int, s *goquery.Selection) {
                if src, ok := s.Attr("src"); ok && src != "" {
                        canonical, relPath, err := localRel(src)
                        if err == nil && relPath != "" {
                                s.SetAttr("src", relPath)
                        }
                        if canonical != "" && urlx.IsCrawlable(canonical) {
                                res.Assets = append(res.Assets, canonical)
                        }
                }
                if srcset, ok := s.Attr("srcset"); ok && srcset != "" {
                        newSrcset, urls := rewriteSrcset(srcset, pageURL, pageDir, shouldMirror)
                        if newSrcset != "" {
                                s.SetAttr("srcset", newSrcset)
                        }
                        res.Assets = append(res.Assets, urls...)
                }
        })

        // 5. <source srcset> and <source src>
        doc.Find("source").Each(func(_ int, s *goquery.Selection) {
                if srcset, ok := s.Attr("srcset"); ok && srcset != "" {
                        newSrcset, urls := rewriteSrcset(srcset, pageURL, pageDir, shouldMirror)
                        if newSrcset != "" {
                                s.SetAttr("srcset", newSrcset)
                        }
                        res.Assets = append(res.Assets, urls...)
                }
                if src, ok := s.Attr("src"); ok && src != "" {
                        canonical, relPath, err := localRel(src)
                        if err == nil && relPath != "" {
                                s.SetAttr("src", relPath)
                        }
                        if canonical != "" && urlx.IsCrawlable(canonical) {
                                res.Assets = append(res.Assets, canonical)
                        }
                }
        })

        // 6. Media: <video src>, <audio src>, <video poster="...">
        doc.Find("video[src], audio[src], video[poster]").Each(func(_ int, s *goquery.Selection) {
                for _, attr := range []string{"src", "poster"} {
                        if val, ok := s.Attr(attr); ok && val != "" {
                                canonical, relPath, err := localRel(val)
                                if err == nil && relPath != "" {
                                        s.SetAttr(attr, relPath)
                                }
                                if canonical != "" && urlx.IsCrawlable(canonical) {
                                        res.Assets = append(res.Assets, canonical)
                                }
                        }
                }
        })

        // 6b. <track src> — subtitle/caption tracks for <video>/<audio>.
        // Without this the mirrored media player has no captions.
        doc.Find("track[src]").Each(func(_ int, s *goquery.Selection) {
                val, _ := s.Attr("src")
                canonical, relPath, err := localRel(val)
                if err == nil && relPath != "" {
                        s.SetAttr("src", relPath)
                }
                if canonical != "" && urlx.IsCrawlable(canonical) {
                        res.Assets = append(res.Assets, canonical)
                }
        })

        // 7. <embed src>, <object data>
        doc.Find("embed[src]").Each(func(_ int, s *goquery.Selection) {
                val, _ := s.Attr("src")
                canonical, relPath, err := localRel(val)
                if err == nil && relPath != "" {
                        s.SetAttr("src", relPath)
                }
                if canonical != "" && urlx.IsCrawlable(canonical) {
                        res.Assets = append(res.Assets, canonical)
                }
        })
        doc.Find("object[data]").Each(func(_ int, s *goquery.Selection) {
                val, _ := s.Attr("data")
                canonical, relPath, err := localRel(val)
                if err == nil && relPath != "" {
                        s.SetAttr("data", relPath)
                }
                if canonical != "" && urlx.IsCrawlable(canonical) {
                        res.Assets = append(res.Assets, canonical)
                }
        })

        // 8. Inline style="background: url(...)" on any element
        rewriteInlineStyles(doc, pageURL, pageDir, shouldMirror, &res.Assets)

        // 9. <style> blocks: rewrite CSS url() and @import
        doc.Find("style").Each(func(_ int, s *goquery.Selection) {
                text := s.Text()
                rewritten, urls := RewriteCSS(text, pageURL, pageDir, shouldMirror)
                if rewritten != text {
                        s.SetText(rewritten)
                }
                res.Assets = append(res.Assets, urls...)
        })

        // 10. <meta http-equiv="refresh" content="0; url=..."> - treat as a page link
        doc.Find(`meta[http-equiv="refresh"]`).Each(func(_ int, s *goquery.Selection) {
                content, ok := s.Attr("content")
                if !ok {
                        return
                }
                // format: "0; url=https://example.com/page"
                if i := strings.Index(strings.ToLower(content), "url="); i >= 0 {
                        target := strings.TrimSpace(content[i+4:])
                        canonical, relPath, err := localRel(target)
                        if err == nil && relPath != "" {
                                newContent := content[:i+4] + relPath
                                s.SetAttr("content", newContent)
                        }
                        if canonical != "" && urlx.IsCrawlable(canonical) {
                                res.PageLinks = append(res.PageLinks, canonical)
                        }
                }
        })

        // Serialize the modified document back to HTML.
        out, err := doc.Html()
        if err != nil {
                return nil, fmt.Errorf("serialize html %s: %w", pageURL, err)
        }
        res.HTML = []byte(out)
        return res, nil
}

// rewriteSrcset parses a srcset attribute value and rewrites each URL to its
// local mirror path. Returns the rewritten srcset and the list of asset URLs.
//
// srcset format: "url1 1x, url2 2x" or "url1 100w, url2 200w"
func rewriteSrcset(srcset, pageURL, pageDir string, shouldMirror func(string) bool) (string, []string) {
        var (
                newParts []string
                urls     []string
        )
        entries := strings.Split(srcset, ",")
        for _, entry := range entries {
                entry = strings.TrimSpace(entry)
                if entry == "" {
                        continue
                }
                // Split URL and descriptor
                fields := strings.Fields(entry)
                if len(fields) == 0 {
                        continue
                }
                rawURL := fields[0]
                descriptor := ""
                if len(fields) > 1 {
                        descriptor = strings.Join(fields[1:], " ")
                }
                canonical, err := urlx.Canonicalize(rawURL, pageURL)
                if err != nil || !urlx.IsCrawlable(canonical) || !shouldMirror(canonical) {
                        newParts = append(newParts, entry)
                        continue
                }
                targetPath, err := urlx.MirrorPath(canonical)
                if err != nil {
                        newParts = append(newParts, entry)
                        continue
                }
                rel := urlx.RelativePath(pageDir, targetPath)
                if descriptor != "" {
                        newParts = append(newParts, rel+" "+descriptor)
                } else {
                        newParts = append(newParts, rel)
                }
                urls = append(urls, canonical)
        }
        return strings.Join(newParts, ", "), urls
}

// rewriteInlineStyles walks every element that has a style="" attribute and
// rewrites any url(...) reference found inside.
func rewriteInlineStyles(doc *goquery.Document, pageURL, pageDir string, shouldMirror func(string) bool, assets *[]string) {
        doc.Find("[style]").Each(func(_ int, s *goquery.Selection) {
                style, ok := s.Attr("style")
                if !ok || style == "" {
                        return
                }
                rewritten, urls := RewriteCSS(style, pageURL, pageDir, shouldMirror)
                if rewritten != style {
                        s.SetAttr("style", rewritten)
                }
                *assets = append(*assets, urls...)
        })
}

// ParseHTMLLinksFallback is a lower-fidelity fallback used when goquery fails
// to parse the document (e.g. broken HTML). It uses a regex-based scan for
// href=/src= attributes. The output is only the discovered URLs, no
// rewriting is performed.
//
// This is intended as a safety net for very malformed HTML; the main path
// should always be RewriteHTML.
func ParseHTMLLinksFallback(body []byte, baseURL string) (links []string) {
        hrefRe := regexp.MustCompile(`(?:href|src)\s*=\s*["']([^"']+)["']`)
        matches := hrefRe.FindAllSubmatch(body, -1)
        for _, m := range matches {
                if len(m) < 2 {
                        continue
                }
                raw := string(m[1])
                if !urlx.IsCrawlable(raw) {
                        continue
                }
                canonical, err := urlx.Canonicalize(raw, baseURL)
                if err != nil {
                        continue
                }
                links = append(links, canonical)
        }
        return links
}

// WalkAttributes can be used by tests or by callers that want the raw
// attribute values before rewriting. It is currently unused at runtime
// but exposed for completeness.
func WalkAttributes(body []byte, pageURL string, cb func(tag, attr, value string)) error {
        node, err := html.Parse(bytes.NewReader(body))
        if err != nil {
                return err
        }
        var walk func(*html.Node)
        walk = func(n *html.Node) {
                if n.Type == html.ElementNode {
                        for _, a := range n.Attr {
                                cb(n.Data, a.Key, a.Val)
                        }
                }
                for c := n.FirstChild; c != nil; c = c.NextSibling {
                        walk(c)
                }
        }
        walk(node)
        return nil
}

// urlParse is a small helper kept here so we don't accidentally shadow the
// net/url package with our local `url` variable names above.
var _ = url.Parse
