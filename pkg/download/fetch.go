// Package download handles HTTP fetching and writing fetched content to the
// local mirror directory.
package download

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Fetcher is a reusable HTTP client tuned for site mirroring. It is safe
// for concurrent use.
type Fetcher struct {
	Client    *http.Client
	UserAgent string
}

// NewFetcher builds a Fetcher with the supplied configuration.
//
//   - timeout: per-request timeout (0 = no timeout).
//   - userAgent: User-Agent header sent with every request.
//   - proxy: optional proxy URL (http or socks5). Empty = direct.
//   - insecureTLS: skip TLS certificate verification (useful for sites with
//     self-signed certs; defaults to false in production).
//   - jar: shared cookie jar (may be nil).
func NewFetcher(timeout time.Duration, userAgent, proxy string, insecureTLS bool, jar *cookiejar.Jar) (*Fetcher, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecureTLS}
	if proxy != "" {
		pu, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy %q: %w", proxy, err)
		}
		transport.Proxy = http.ProxyURL(pu)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	if jar != nil {
		client.Jar = jar
	}

	if userAgent == "" {
		userAgent = "Mozilla/5.0 (compatible; WebClone/1.0; +https://github.com/a-talebifard/webclone)"
	}
	return &Fetcher{Client: client, UserAgent: userAgent}, nil
}

// FetchResult holds the metadata + raw body of a fetched URL.
type FetchResult struct {
	URL         string
	StatusCode  int
	ContentType string
	Body        []byte
	FromCache   bool
}

// Fetch performs an HTTP GET on the given URL with the configured client.
// It returns the response body bytes (already consumed and closed).
//
// A non-2xx status code is returned as an error containing the status.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "identity") // no gzip - we want raw bytes for rewriting

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", rawURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// For 4xx/5xx we still return the body so the caller can decide what
		// to do with it. The error includes the status for filtering.
		return &FetchResult{
			URL:         rawURL,
			StatusCode:  resp.StatusCode,
			ContentType: resp.Header.Get("Content-Type"),
			Body:        body,
		}, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}

	return &FetchResult{
		URL:         rawURL,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
	}, nil
}

// SaveAtomic writes data to dstPath, creating parent directories as needed.
// It is safe for concurrent use as long as no two goroutines write to the
// SAME path simultaneously (which the crawler's visited-set prevents).
//
// If a file already exists at dstPath, it is overwritten.
func SaveAtomic(dstPath string, data []byte) error {
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	// Write to a temp file in the same directory then rename, to make the
	// update atomic-ish.
	tmp := dstPath + ".webclone-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s: %w", dstPath, err)
	}
	return nil
}

// FileExists returns true if the given path exists and is a regular file.
func FileExists(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// dirMu serializes directory creation for the same parent dir to avoid
// the race where two goroutines try MkdirAll the same path at once.
// (MkdirAll actually handles this gracefully, but it still logs errors
// when called concurrently; the per-dir lock keeps things quiet.)
var dirMu sync.Map // map[string]*sync.Mutex

func lockForDir(dir string) *sync.Mutex {
	v, _ := dirMu.LoadOrStore(dir, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// SaveWithDirLock is a variant of SaveAtomic that also takes a per-dir
// lock to avoid interleaved writes when many workers hit the same
// directory simultaneously. Useful for shared asset directories.
func SaveWithDirLock(dstPath string, data []byte) error {
	mu := lockForDir(filepath.Dir(dstPath))
	mu.Lock()
	defer mu.Unlock()
	return SaveAtomic(dstPath, data)
}

// ErrSkipped indicates a URL was deliberately skipped by policy.
var ErrSkipped = errors.New("skipped by policy")
