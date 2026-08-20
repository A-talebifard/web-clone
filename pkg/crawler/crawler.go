// Package crawler implements the BFS site crawler that downloads pages and
// assets concurrently while preserving the URL→filesystem mirror mapping.
//
// # Design
//
// The crawler maintains:
//
//   - a work channel (URLs to process)
//   - a sync.Map visited set (URLs already enqueued/done)
//   - an in-flight counter (so we know when crawling has finished)
//
// Each worker:
//
//  1. Pops a URL from the channel.
//  2. Fetches it (HEAD not used - many sites don't support HEAD well).
//  3. Saves the raw bytes to the mirror path.
//  4. If HTML, parses it, rewrites links to local mirror paths, saves the
//     rewritten HTML, and enqueues page links + asset URLs.
//  5. If CSS, parses it, rewrites url()/@import, saves rewritten CSS, and
//     enqueues referenced asset URLs.
//  6. Otherwise (binary asset), the raw bytes saved in step 3 are the
//     final result; nothing else to do.
//
// Concurrency is bounded by Config.Workers. A safety max-URLs cap
// (Config.MaxURLs) protects against infinite crawls when the user opts
// into "all links unlimited" on a site with link loops.
package crawler

import (
        "context"
        "fmt"
        "net/http/cookiejar"
        "net/url"
        "os"
        "path"
        "path/filepath"
        "strings"
        "sync"
        "sync/atomic"
        "time"

        "github.com/a-talebifard/webclone/pkg/download"
        "github.com/a-talebifard/webclone/pkg/parse"
        "github.com/a-talebifard/webclone/pkg/urlx"
)

// Config holds all tunables for a crawl run.
type Config struct {
        // OutputDir is the directory that will contain the mirror tree.
        // e.g. "./out" - the mirror root. URLs from example.com will be written
        // to "./out/example.com/...".
        OutputDir string

        // Workers is the max number of concurrent fetch workers.
        Workers int

        // MaxURLs is a hard cap on the number of distinct URLs the crawler will
        // process. 0 = unlimited (still bounded by MaxDepth + visited set).
        MaxURLs int

        // MaxDepth is the max number of link-hops from the seed URL.
        // 0 = unlimited.
        MaxDepth int

        // SameDomainOnly restricts the crawler to URLs whose registered domain
        // matches the seed's registered domain. When false (the default),
        // external links are also crawled (subject to MaxURLs/MaxDepth caps).
        SameDomainOnly bool

        // AllowExternalSubdomains works with SameDomainOnly. When true, the
        // crawler also follows subdomains of the seed domain.
        // e.g. seed example.com -> www.example.com, blog.example.com are OK.
        AllowExternalSubdomains bool

        // AllowedHosts is an explicit allowlist of hosts the crawler may
        // crawl. When non-empty, SameDomainOnly is ignored and only these
        // hosts (case-insensitive) are followed.
        AllowedHosts []string

        // AssetExtensions is an optional allowlist of file extensions to
        // download as static assets (case-insensitive, with leading dot).
        // When empty, every asset URL referenced by HTML/CSS is downloaded.
        // Use this to skip heavy assets like video files.
        AssetExtensions []string

        // SkipAssets, if true, downloads ONLY HTML pages - no CSS/JS/images.
        SkipAssets bool

        // Timeout per HTTP request. 0 = no timeout.
        Timeout time.Duration

        // UserAgent sent with every request. Empty -> default.
        UserAgent string

        // Proxy URL (http://, https://, socks5://). Empty -> direct.
        Proxy string

        // InsecureTLS skips TLS cert verification.
        InsecureTLS bool

        // CookieJar shared across all requests. May be nil.
        CookieJar *cookiejar.Jar

        // Verbose controls whether per-URL log lines are printed.
        Verbose bool

        // SaveManifest writes a manifest.json file with all crawled URLs.
        SaveManifest bool

        // PreserveQueryInPath keeps the query string when computing the local
        // mirror path (currently always enabled via urlx - kept for future).
        PreserveQueryInPath bool

        // Events is an optional event emitter. When non-nil, the crawler
        // publishes ProgressEvents on every meaningful state change
        // (fetch start/ok/error, save, enqueue, etc.). GUIs use this to
        // update live progress views without polling.
        Events *EventEmitter

        // JobIndex, JobTotal and JobURL are optional metadata that get stamped
        // onto every emitted ProgressEvent. They have no effect on crawl
        // behavior; they exist so a job-queue orchestrator (like the webclone
        // GUI) can label events with "this is job 2 of 5, crawling
        // https://example.com" without the GUI having to track that state
        // itself. Leave all zero/empty when running a single crawl.
        JobIndex int
        JobTotal int
        JobURL   string
}

// Stats holds counters reported during and after the crawl.
type Stats struct {
        Visited    atomic.Int64
        Downloaded atomic.Int64
        Failed     atomic.Int64
        Skipped    atomic.Int64
        Bytes      atomic.Int64
        Pages      atomic.Int64
        Assets     atomic.Int64
        StartTime  time.Time
        EndTime    time.Time
}

// Crawler is the orchestrator. Construct with New().
type Crawler struct {
        cfg     Config
        fetch   *download.Fetcher
        stats   Stats
        queue   chan string
        wg      sync.WaitGroup // counts in-flight URLs (enqueued but not processed)
        mu      sync.Mutex
        visited sync.Map // map[string]struct{}
        seed    *url.URL

        // pauseMu guards the pause/resume mechanism. When the crawler is paused,
        // every worker blocks on pauseCh until Resume is called (which replaces
        // pauseCh with an already-closed channel, unblocking all waiters).
        pauseMu  sync.Mutex
        pauseCh  chan struct{}
        paused   bool
        cancelFn context.CancelFunc

        // logf is used so tests can capture logs.
        logf func(format string, args ...any)
}

// New constructs a Crawler with the given configuration. The crawler does
// not start fetching until Run is called.
func New(cfg Config) (*Crawler, error) {
        if cfg.OutputDir == "" {
                return nil, fmt.Errorf("OutputDir is required")
        }
        if cfg.Workers <= 0 {
                cfg.Workers = 5
        }
        if cfg.Timeout == 0 {
                cfg.Timeout = 60 * time.Second
        }

        fetcher, err := download.NewFetcher(
                cfg.Timeout, cfg.UserAgent, cfg.Proxy,
                cfg.InsecureTLS, cfg.CookieJar,
        )
        if err != nil {
                return nil, err
        }

        c := &Crawler{
                cfg:     cfg,
                fetch:   fetcher,
                queue:   make(chan string, 4096),
                logf: func(format string, args ...any) {
                        if cfg.Verbose {
                                fmt.Printf(format+"\n", args...)
                        }
                },
                // Start in the running state: an already-closed pauseCh means
                // workers never block waiting for a resume.
                pauseCh: closedCh(),
        }
        return c, nil
}

// closedCh returns a channel that is already closed. Workers waiting on it
// proceed immediately, which is the desired behavior when the crawler is not
// paused.
func closedCh() chan struct{} {
        ch := make(chan struct{})
        close(ch)
        return ch
}

// Run starts the crawler and blocks until all reachable URLs (within the
// configured caps) have been processed, or until ctx is canceled.
//
// seedURL is the URL the crawl starts from.
func (c *Crawler) Run(ctx context.Context, seedURL string) (*Stats, error) {
        if !urlx.IsCrawlable(seedURL) {
                return nil, fmt.Errorf("seed URL is not crawlable: %s", seedURL)
        }
        canonicalSeed, err := urlx.Canonicalize(seedURL, seedURL)
        if err != nil {
                return nil, fmt.Errorf("canonicalize seed: %w", err)
        }
        seed, err := url.Parse(canonicalSeed)
        if err != nil {
                return nil, fmt.Errorf("parse seed: %w", err)
        }
        c.seed = seed

        if err := os.MkdirAll(c.cfg.OutputDir, 0o755); err != nil {
                return nil, fmt.Errorf("mkdir output dir: %w", err)
        }

        c.stats.StartTime = time.Now()

        // Print initial banner
        fmt.Printf("webclone: crawling %s (workers=%d, maxURLs=%d, maxDepth=%d, sameDomain=%v)\n",
                canonicalSeed, c.cfg.Workers, c.cfg.MaxURLs, c.cfg.MaxDepth, c.cfg.SameDomainOnly)

        // Emit start event for GUIs
        c.emit(ProgressEvent{
                Type: EventStart,
                Time: c.stats.StartTime,
                URL:  canonicalSeed,
                Msg:  fmt.Sprintf("crawling %s (workers=%d, maxURLs=%d, maxDepth=%d)", canonicalSeed, c.cfg.Workers, c.cfg.MaxURLs, c.cfg.MaxDepth),
        })

        // Start workers
        workerCtx, cancel := context.WithCancel(ctx)
        defer cancel()
        c.cancelFn = cancel

        for i := 0; i < c.cfg.Workers; i++ {
                go c.worker(workerCtx, i)
        }

        // Enqueue seed
        c.enqueue(canonicalSeed, 0)

        // Wait for everything to finish, then close the queue and wait for workers.
        c.wg.Wait()
        close(c.queue)

        c.stats.EndTime = time.Now()

        if c.cfg.SaveManifest {
                if err := c.writeManifest(); err != nil {
                        fmt.Printf("warning: failed to write manifest: %v\n", err)
                }
        }

        c.emit(ProgressEvent{
                Type: EventEnd,
                Time: c.stats.EndTime,
                Msg:  fmt.Sprintf("done: %d pages, %d assets, %d failed", c.stats.Pages.Load(), c.stats.Assets.Load(), c.stats.Failed.Load()),
        })
        return &c.stats, nil
}

// RunSeeds is a convenience wrapper that crawls multiple seed URLs in
// sequence using the same crawler instance and the same output directory.
// Each seed is enqueued at depth 0. The visited-set is shared across all
// seeds so URLs discovered for one seed that are also reachable from another
// seed are only fetched once.
//
// This is the multi-site ("batch download") entry point. The crawler's
// domain policy (SameDomainOnly / AllowedHosts) is applied per-URL using
// the FIRST seed as the reference host. When crawling multiple unrelated
// sites in one batch, the caller should leave SameDomainOnly=false OR pass
// an explicit AllowedHosts list that covers every seed's host.
func (c *Crawler) RunSeeds(ctx context.Context, seeds []string) (*Stats, error) {
        if len(seeds) == 0 {
                return nil, fmt.Errorf("no seed URLs provided")
        }
        // Validate and canonicalize every seed up front so we fail fast on
        // bad input rather than midway through a long batch.
        canonicalSeeds := make([]string, 0, len(seeds))
        for _, s := range seeds {
                if !urlx.IsCrawlable(s) {
                        return nil, fmt.Errorf("seed URL is not crawlable: %s", s)
                }
                cs, err := urlx.Canonicalize(s, s)
                if err != nil {
                        return nil, fmt.Errorf("canonicalize seed %s: %w", s, err)
                }
                canonicalSeeds = append(canonicalSeeds, cs)
        }

        // Use the first seed as the reference for domain policy. The caller
        // is responsible for ensuring the policy is compatible with every
        // seed (e.g. by setting AllowedHosts explicitly).
        firstSeed, err := url.Parse(canonicalSeeds[0])
        if err != nil {
                return nil, fmt.Errorf("parse seed: %w", err)
        }
        c.seed = firstSeed

        if err := os.MkdirAll(c.cfg.OutputDir, 0o755); err != nil {
                return nil, fmt.Errorf("mkdir output dir: %w", err)
        }

        c.stats.StartTime = time.Now()

        fmt.Printf("webclone: batch crawling %d seed(s) (workers=%d, maxURLs=%d, maxDepth=%d, sameDomain=%v)\n",
                len(canonicalSeeds), c.cfg.Workers, c.cfg.MaxURLs, c.cfg.MaxDepth, c.cfg.SameDomainOnly)

        c.emit(ProgressEvent{
                Type: EventStart,
                Time: c.stats.StartTime,
                URL:  canonicalSeeds[0],
                Msg:  fmt.Sprintf("batch crawling %d seeds (workers=%d, maxURLs=%d, maxDepth=%d)", len(canonicalSeeds), c.cfg.Workers, c.cfg.MaxURLs, c.cfg.MaxDepth),
        })

        workerCtx, cancel := context.WithCancel(ctx)
        defer cancel()
        c.cancelFn = cancel

        for i := 0; i < c.cfg.Workers; i++ {
                go c.worker(workerCtx, i)
        }

        for _, s := range canonicalSeeds {
                c.enqueue(s, 0)
        }

        c.wg.Wait()
        close(c.queue)

        c.stats.EndTime = time.Now()

        if c.cfg.SaveManifest {
                if err := c.writeManifest(); err != nil {
                        fmt.Printf("warning: failed to write manifest: %v\n", err)
                }
        }

        c.emit(ProgressEvent{
                Type: EventEnd,
                Time: c.stats.EndTime,
                Msg:  fmt.Sprintf("done: %d pages, %d assets, %d failed", c.stats.Pages.Load(), c.stats.Assets.Load(), c.stats.Failed.Load()),
        })
        return &c.stats, nil
}

// emit publishes a progress event to all subscribers (if an emitter is
// configured). It snapshots the stats counters into the event so GUIs get
// a consistent view. Safe to call from any goroutine.
func (c *Crawler) emit(ev ProgressEvent) {
        if c.cfg.Events == nil {
                return
        }
        if ev.Time.IsZero() {
                ev.Time = time.Now()
        }
        // Snapshot the atomic counters into the event payload.
        ev.Visited = c.stats.Visited.Load()
        ev.Downloaded = c.stats.Downloaded.Load()
        ev.Failed = c.stats.Failed.Load()
        ev.Skipped = c.stats.Skipped.Load()
        ev.TotalBytes = c.stats.Bytes.Load()
        ev.Pages = c.stats.Pages.Load()
        ev.Assets = c.stats.Assets.Load()
        ev.StartTime = c.stats.StartTime
        ev.JobIndex = c.cfg.JobIndex
        ev.JobTotal = c.cfg.JobTotal
        ev.JobURL = c.cfg.JobURL
        c.cfg.Events.Emit(ev)
}

// SnapshotCounters returns the live counter values. This is what the GUI
// should call when it wants a consistent snapshot of all counters.
func (c *Crawler) SnapshotCounters() (visited, downloaded, failed, skipped, bytes, pages, assets int64) {
        return c.stats.Visited.Load(),
                c.stats.Downloaded.Load(),
                c.stats.Failed.Load(),
                c.stats.Skipped.Load(),
                c.stats.Bytes.Load(),
                c.stats.Pages.Load(),
                c.stats.Assets.Load()
}

// enqueue adds a URL to the work queue if it has not already been visited
// and if the global caps have not been exceeded. Increments the in-flight
// counter so the crawler knows the URL is pending.
//
// depth is the link-hop distance from the seed. depth=0 means "this is the
// seed".  We track depth per-URL but enqueue it as part of the URL string
// (encoded in a sidecar map) to keep the queue type simple.
func (c *Crawler) enqueue(rawURL string, depth int) {
        if !urlx.IsCrawlable(rawURL) {
                return
        }
        if c.cfg.MaxURLs > 0 && c.stats.Visited.Load() >= int64(c.cfg.MaxURLs) {
                c.stats.Skipped.Add(1)
                return
        }
        if c.cfg.MaxDepth > 0 && depth > c.cfg.MaxDepth {
                c.stats.Skipped.Add(1)
                return
        }
        if !c.shouldCrawl(rawURL) {
                c.stats.Skipped.Add(1)
                return
        }

        // Deduplicate
        if _, ok := c.visited.LoadOrStore(rawURL, depth); ok {
                // already seen - check if we have a higher depth recorded; if so,
                // update so we don't skip a shorter path.
                // (We keep the original depth - BFS guarantees first encounter is
                // shortest path.)
                return
        }

        // Respect maxURLs: visited counts every URL we've enqueued.
        if c.cfg.MaxURLs > 0 && c.stats.Visited.Load() > int64(c.cfg.MaxURLs) {
                return
        }
        c.stats.Visited.Add(1)

        c.wg.Add(1)
        select {
        case c.queue <- rawURL:
        case <-time.After(60 * time.Second):
                // queue is full and stuck; bail to avoid deadlock
                c.wg.Done()
                fmt.Printf("warning: queue full, dropping %s\n", rawURL)
        }
}

// shouldCrawl decides whether a given canonical URL is within the policy
// (domain restrictions, host allowlist, asset extension filter).
func (c *Crawler) shouldCrawl(rawURL string) bool {
        u, err := url.Parse(rawURL)
        if err != nil {
                return false
        }
        if !urlx.IsHTTPScheme(u) {
                return false
        }

        // Host-based policy
        if len(c.cfg.AllowedHosts) > 0 {
                host := strings.ToLower(u.Hostname())
                ok := false
                for _, h := range c.cfg.AllowedHosts {
                        if host == strings.ToLower(h) {
                                ok = true
                                break
                        }
                }
                if !ok {
                        return false
                }
        } else if c.cfg.SameDomainOnly && c.seed != nil {
                host := u.Hostname()
                seedHost := c.seed.Hostname()
                if c.cfg.AllowExternalSubdomains {
                        // host must end with "." + seedHost OR equal seedHost
                        if host != seedHost && !strings.HasSuffix(host, "."+seedHost) {
                                return false
                        }
                } else {
                        if host != seedHost {
                                return false
                        }
                }
        }

        // Asset extension filter
        if len(c.cfg.AssetExtensions) > 0 {
                ext := strings.ToLower(path.Ext(u.Path))
                if ext != "" {
                        allowed := false
                        for _, e := range c.cfg.AssetExtensions {
                                if ext == strings.ToLower(e) {
                                        allowed = true
                                        break
                                }
                        }
                        if !allowed {
                                return false
                        }
                }
        }

        return true
}

// shouldMirror is the same check as shouldCrawl but called from inside the
// HTML rewriter to decide whether to rewrite an attribute to a local path.
// If we won't crawl the URL, we don't rewrite the attribute either - we
// leave it pointing at the original URL.
func (c *Crawler) shouldMirror(rawURL string) bool {
        return c.shouldCrawl(rawURL)
}

// worker is the main per-URL processing loop.
func (c *Crawler) worker(ctx context.Context, id int) {
        for {
                // If the crawler is paused, block here until Resume is called
                // (or the context is cancelled). This means a pause takes effect
                // between URLs, never mid-fetch.
                if err := c.waitIfPaused(ctx); err != nil {
                        return
                }
                select {
                case <-ctx.Done():
                        return
                case rawURL, ok := <-c.queue:
                        if !ok {
                                return
                        }
                        c.process(ctx, rawURL)
                        c.wg.Done()
                }
        }
}

// process is the per-URL pipeline: fetch -> save -> parse -> enqueue children.
func (c *Crawler) process(ctx context.Context, rawURL string) {
        depth := 0
        if v, ok := c.visited.Load(rawURL); ok {
                if d, ok := v.(int); ok {
                        depth = d
                }
        }

        c.logf("[%5d] fetching %s", c.stats.Downloaded.Load()+1, rawURL)
        c.emit(ProgressEvent{Type: EventFetchStart, URL: rawURL, Msg: "fetching"})

        result, err := c.fetch.Fetch(ctx, rawURL)
        if err != nil {
                c.stats.Failed.Add(1)
                c.emit(ProgressEvent{Type: EventFetchError, URL: rawURL, Msg: err.Error()})
                // If we have a partial body, still try to save what we have.
                if result == nil || len(result.Body) == 0 {
                        fmt.Printf("error: %v\n", err)
                        return
                }
        }
        if result == nil {
                return
        }

        c.stats.Downloaded.Add(1)
        c.stats.Bytes.Add(int64(len(result.Body)))
        c.emit(ProgressEvent{Type: EventFetchOK, URL: rawURL, Bytes: int64(len(result.Body))})

        // Compute local mirror path
        mirrorPath, err := urlx.MirrorPath(rawURL)
        if err != nil {
                c.stats.Failed.Add(1)
                fmt.Printf("error: mirror path for %s: %v\n", rawURL, err)
                return
        }
        dstPath := filepath.Join(c.cfg.OutputDir, mirrorPath)

        contentType := strings.ToLower(result.ContentType)
        isHTML := urlx.IsHTMLAsset(rawURL) || strings.Contains(contentType, "html")
        isCSS := strings.HasSuffix(strings.ToLower(mirrorPath), ".css")

        if isHTML {
                c.stats.Pages.Add(1)
                // Parse and rewrite HTML
                rewritten, err := parse.RewriteHTML(result.Body, rawURL, c.shouldMirror)
                if err != nil {
                        // fallback: save raw HTML
                        c.logf("warning: html parse failed for %s: %v - saving raw", rawURL, err)
                        if err := download.SaveWithDirLock(dstPath, result.Body); err != nil {
                                fmt.Printf("error: save %s: %v\n", dstPath, err)
                                c.stats.Failed.Add(1)
                                return
                        }
                        // still extract some links via fallback regex
                        links := parse.ParseHTMLLinksFallback(result.Body, rawURL)
                        for _, l := range links {
                                c.enqueue(l, depth+1)
                        }
                        return
                }
                if err := download.SaveWithDirLock(dstPath, rewritten.HTML); err != nil {
                        fmt.Printf("error: save %s: %v\n", dstPath, err)
                        c.stats.Failed.Add(1)
                        return
                }
                c.logf("  saved %s (%d bytes, %d links, %d assets)",
                        dstPath, len(rewritten.HTML), len(rewritten.PageLinks), len(rewritten.Assets))
                c.emit(ProgressEvent{Type: EventSave, URL: rawURL, Path: dstPath, Bytes: int64(len(rewritten.HTML)), Msg: "page"})
                // Enqueue page links
                for _, l := range dedupStrings(rewritten.PageLinks) {
                        c.enqueue(l, depth+1)
                }
                // Enqueue assets
                if !c.cfg.SkipAssets {
                        for _, a := range dedupStrings(rewritten.Assets) {
                                c.enqueue(a, depth+1)
                        }
                }
                return
        }

        if isCSS {
                c.stats.Assets.Add(1)
                pageDir, _ := urlx.MirrorDir(rawURL)
                rewritten, urls := parse.RewriteCSS(string(result.Body), rawURL, pageDir, c.shouldMirror)
                if err := download.SaveWithDirLock(dstPath, []byte(rewritten)); err != nil {
                        fmt.Printf("error: save css %s: %v\n", dstPath, err)
                        c.stats.Failed.Add(1)
                        return
                }
                c.logf("  saved %s (%d bytes, %d css refs)", dstPath, len(rewritten), len(urls))
                c.emit(ProgressEvent{Type: EventSave, URL: rawURL, Path: dstPath, Bytes: int64(len(rewritten)), Msg: "css"})
                if !c.cfg.SkipAssets {
                        for _, u := range dedupStrings(urls) {
                                c.enqueue(u, depth+1)
                        }
                }
                return
        }

        // Binary asset: just save the raw bytes.
        c.stats.Assets.Add(1)
        if err := download.SaveWithDirLock(dstPath, result.Body); err != nil {
                fmt.Printf("error: save %s: %v\n", dstPath, err)
                c.stats.Failed.Add(1)
                return
        }
        c.logf("  saved %s (%d bytes)", dstPath, len(result.Body))
        c.emit(ProgressEvent{Type: EventSave, URL: rawURL, Path: dstPath, Bytes: int64(len(result.Body)), Msg: "asset"})
}

// writeManifest dumps a list of every URL we visited (regardless of
// success/failure) into manifest.json inside the output dir.
func (c *Crawler) writeManifest() error {
        type entry struct {
                URL   string `json:"url"`
                Depth int    `json:"depth"`
        }
        var entries []entry
        c.visited.Range(func(k, v any) bool {
                entries = append(entries, entry{URL: k.(string), Depth: v.(int)})
                return true
        })

        // Build JSON manually so we don't pull in encoding/json for just this.
        // (We do use encoding/json for the manifest, actually.)
        manifestPath := filepath.Join(c.cfg.OutputDir, "manifest.json")
        var b strings.Builder
        b.WriteString("{\n  \"seed\": ")
        if c.seed != nil {
                b.WriteString(fmt.Sprintf("%q", c.seed.String()))
        } else {
                b.WriteString("null")
        }
        b.WriteString(",\n  \"urls\": [\n")
        for i, e := range entries {
                if i > 0 {
                        b.WriteString(",\n")
                }
                b.WriteString(fmt.Sprintf("    {\"url\": %q, \"depth\": %d}", e.URL, e.Depth))
        }
        b.WriteString("\n  ]\n}\n")
        return os.WriteFile(manifestPath, []byte(b.String()), 0o644)
}

// dedupStrings returns the input with duplicates removed, preserving order.
func dedupStrings(in []string) []string {
        if len(in) <= 1 {
                return in
        }
        seen := make(map[string]struct{}, len(in))
        out := make([]string, 0, len(in))
        for _, s := range in {
                if _, ok := seen[s]; ok {
                        continue
                }
                seen[s] = struct{}{}
                out = append(out, s)
        }
        return out
}

// Pause suspends every worker until Resume is called. Workers that are
// currently in the middle of an HTTP fetch will finish that fetch; the pause
// takes effect before they pull the next URL off the queue. Calling Pause
// when already paused is a no-op.
func (c *Crawler) Pause() {
        c.pauseMu.Lock()
        defer c.pauseMu.Unlock()
        if c.paused {
                return
        }
        c.paused = true
        c.pauseCh = make(chan struct{})
        c.emit(ProgressEvent{Type: EventPause, Msg: "crawl paused"})
}

// Resume unblocks all workers paused by a prior Pause call. Calling Resume
// when not paused is a no-op.
func (c *Crawler) Resume() {
        c.pauseMu.Lock()
        defer c.pauseMu.Unlock()
        if !c.paused {
                return
        }
        c.paused = false
        close(c.pauseCh)
        c.emit(ProgressEvent{Type: EventResume, Msg: "crawl resumed"})
}

// IsPaused returns whether the crawler is currently paused.
func (c *Crawler) IsPaused() bool {
        c.pauseMu.Lock()
        defer c.pauseMu.Unlock()
        return c.paused
}

// waitIfPaused blocks until the crawler is resumed. If the crawler is not
// paused, it returns immediately. Workers call this before pulling the next
// URL off the queue so a pause takes effect between URLs, never mid-fetch.
func (c *Crawler) waitIfPaused(ctx context.Context) error {
        c.pauseMu.Lock()
        ch := c.pauseCh
        c.pauseMu.Unlock()
        select {
        case <-ch:
                return nil
        case <-ctx.Done():
                return ctx.Err()
        }
}
