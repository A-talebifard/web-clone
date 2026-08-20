// Package webui implements a browser-based GUI for webclone.
//
// Instead of using a native GUI toolkit (which would require CGO and
// platform-specific dependencies), this package serves a local HTTP
// server that the user opens in their default browser. The browser
// renders the UI; the Go backend handles crawler orchestration and
// pushes live updates back to the browser via Server-Sent Events.
//
// Architecture:
//
//      ┌────────────┐   HTTP/JSON   ┌──────────────┐
//      │  Browser   │ ────────────> │  webui Server │
//      │  (UI/JS)   │ <─── SSE ──── │  (Go backend) │
//      └────────────┘               └──────┬───────┘
//                                             │ config
//                                             ▼
//                                     ┌──────────────┐
//                                     │  crawler.Crawler │
//                                     └──────────────┘
//
// This package has zero CGO dependencies and works on Windows/macOS/Linux
// without any external libraries.
package webui

import (
        "context"
        "embed"
        "encoding/json"
        "fmt"
        "net/http"
        "net/http/cookiejar"
        "net/url"
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strings"
        "sync"
        "time"

        "github.com/a-talebifard/webclone/pkg/crawler"
)

// staticFiles embeds the HTML/CSS/JS files into the binary.
//
//go:embed static/*
var staticFiles embed.FS

// Server is the local HTTP server that serves the GUI + crawler API.
type Server struct {
        server    *http.Server
        crawlerMu sync.Mutex
        crawler   *crawler.Crawler
        cancel    context.CancelFunc
        done      chan struct{}
        emitter   *crawler.EventEmitter

        // Job-queue state. When the GUI calls /api/jobs/start, the server
        // runs every job in `jobs` sequentially. jobQueueMu guards the queue
        // state so /api/status can report "currently running job 2 of 5".
        // The queue is read-only after runJobs is launched; mutation happens
        // only when a new queue is started (which replaces the slice atomically
        // under the lock).
        jobQueueMu sync.RWMutex
        jobs       []Job
        jobIndex   int    // 0-based index of the currently running job
        jobState   string // "idle" | "running" | "paused" | "done" | "stopped"

        // subscribers are SSE clients; each has its own channel.
        subsMu sync.RWMutex
        subs   map[chan crawler.ProgressEvent]struct{}
}

// Job is a single crawl job in a queue. Each job has its own seed URL,
// output directory, and full set of crawler options. The GUI builds a list
// of these (one per site the user wants to mirror) and submits them all at
// once via /api/jobs/start.
type Job struct {
        URL          string `json:"url"`
        OutputDir    string `json:"output_dir"`
        Workers      int    `json:"workers"`
        MaxURLs      int    `json:"max_urls"`
        MaxDepth     int    `json:"max_depth"`
        Timeout      int    `json:"timeout"`
        SameDomain   bool   `json:"same_domain"`
        AllowSubs    bool   `json:"allow_subdomains"`
        SkipAssets   bool   `json:"skip_assets"`
        InsecureTLS  bool   `json:"insecure_tls"`
        Manifest     bool   `json:"manifest"`
        Proxy        string `json:"proxy"`
        UserAgent    string `json:"user_agent"`
        Cookies      string `json:"cookies"`
        AllowedHosts string `json:"allowed_hosts"`
        AssetExt     string `json:"asset_ext"`
}

// New constructs a Server listening on the given port.
func New(port int) *Server {
        s := &Server{
                subs: make(map[chan crawler.ProgressEvent]struct{}),
        }
        mux := http.NewServeMux()
        mux.HandleFunc("/", s.handleIndex)
        mux.HandleFunc("/static/", s.handleStatic)
        mux.HandleFunc("/api/start", s.handleStart)
        mux.HandleFunc("/api/jobs/start", s.handleJobsStart)
        mux.HandleFunc("/api/stop", s.handleStop)
        mux.HandleFunc("/api/pause", s.handlePause)
        mux.HandleFunc("/api/resume", s.handleResume)
        mux.HandleFunc("/api/status", s.handleStatus)
        mux.HandleFunc("/api/events", s.handleEvents)
        mux.HandleFunc("/api/open", s.handleOpen)
        mux.HandleFunc("/api/serve", s.handleServe)
        mux.HandleFunc("/api/browse", s.handleBrowse)

        s.server = &http.Server{
                Addr:    fmt.Sprintf("127.0.0.1:%d", port),
                Handler: mux,
        }
        return s
}

// Start begins listening and serving. It blocks until the server stops.
// OpenBrowser should be called separately (or as a goroutine) to launch
// the user's browser at the right URL.
func (s *Server) Start() error {
        return s.server.ListenAndServe()
}

// OpenBrowser opens the user's default browser at the server's URL.
// This is a no-op if the OS is not recognized.
func (s *Server) OpenBrowser() {
        url := "http://" + s.server.Addr
        // Give the server a moment to start accepting connections before
        // launching the browser.
        go func() {
                time.Sleep(300 * time.Millisecond)
                openInOS(url)
        }()
}

// --- HTTP handlers ---

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
                http.NotFound(w, r)
                return
        }
        data, err := staticFiles.ReadFile("static/index.html")
        if err != nil {
                http.Error(w, "internal error", 500)
                return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write(data)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/static/")
        data, err := staticFiles.ReadFile("static/" + path)
        if err != nil {
                http.NotFound(w, r)
                return
        }
        switch {
        case strings.HasSuffix(path, ".css"):
                w.Header().Set("Content-Type", "text/css; charset=utf-8")
        case strings.HasSuffix(path, ".js"):
                w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
        case strings.HasSuffix(path, ".html"):
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
        case strings.HasSuffix(path, ".svg"):
                w.Header().Set("Content-Type", "image/svg+xml")
        }
        w.Write(data)
}

// startRequest is the JSON payload for /api/start.
//
// URLs is an optional list of additional seed URLs for batch downloads. When
// non-empty, the crawler treats every entry as a seed and crawls them all
// in one run with a shared visited-set. The first URL in `URL` is still
// treated as the primary seed for domain-policy purposes; if you need to
// crawl unrelated hosts in one batch, leave SameDomain=false or set
// AllowedHosts explicitly to cover every seed host.
type startRequest struct {
        URL          string   `json:"url"`
        URLs         []string `json:"urls"` // additional seed URLs for batch mode
        OutputDir    string   `json:"output_dir"`
        Workers      int      `json:"workers"`
        MaxURLs      int      `json:"max_urls"`
        MaxDepth     int      `json:"max_depth"`
        Timeout      int      `json:"timeout"`
        SameDomain   bool     `json:"same_domain"`
        AllowSubs    bool     `json:"allow_subdomains"`
        SkipAssets   bool     `json:"skip_assets"`
        InsecureTLS  bool     `json:"insecure_tls"`
        Manifest     bool     `json:"manifest"`
        Proxy        string   `json:"proxy"`
        UserAgent    string   `json:"user_agent"`
        Cookies      string   `json:"cookies"`
        AllowedHosts string   `json:"allowed_hosts"`
        AssetExt     string   `json:"asset_ext"`
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        var req startRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeJSON(w, 400, map[string]string{"error": err.Error()})
                return
        }
        if req.URL == "" {
                writeJSON(w, 400, map[string]string{"error": "url is required"})
                return
        }
        if req.OutputDir == "" {
                writeJSON(w, 400, map[string]string{"error": "output_dir is required"})
                return
        }

        // Make sure no crawl is already running
        s.crawlerMu.Lock()
        if s.crawler != nil {
                s.crawlerMu.Unlock()
                writeJSON(w, 409, map[string]string{"error": "crawl already running"})
                return
        }
        s.crawlerMu.Unlock()

        // Build crawler config
        cfg := crawler.Config{
                OutputDir:               req.OutputDir,
                Workers:                 req.Workers,
                MaxURLs:                 req.MaxURLs,
                MaxDepth:                req.MaxDepth,
                SameDomainOnly:          req.SameDomain,
                AllowExternalSubdomains: req.AllowSubs,
                SkipAssets:              req.SkipAssets,
                InsecureTLS:             req.InsecureTLS,
                SaveManifest:            req.Manifest,
                Proxy:                   req.Proxy,
                UserAgent:               req.UserAgent,
                Timeout:                 time.Duration(req.Timeout) * time.Second,
        }
        if req.AllowedHosts != "" {
                cfg.AllowedHosts = strings.Split(req.AllowedHosts, ",")
                for i := range cfg.AllowedHosts {
                        cfg.AllowedHosts[i] = strings.TrimSpace(cfg.AllowedHosts[i])
                }
        }
        if req.AssetExt != "" {
                cfg.AssetExtensions = strings.Split(req.AssetExt, ",")
                for i := range cfg.AssetExtensions {
                        cfg.AssetExtensions[i] = strings.TrimSpace(cfg.AssetExtensions[i])
                }
        }

        // Create event emitter and subscribe the SSE broadcaster
        emitter := crawler.NewEventEmitter()
        cfg.Events = emitter
        cfg.Verbose = false // GUI gets logs via events, not stdout

        c, err := crawler.New(cfg)
        if err != nil {
                writeJSON(w, 500, map[string]string{"error": err.Error()})
                return
        }

        ctx, cancel := context.WithCancel(context.Background())
        s.crawlerMu.Lock()
        s.crawler = c
        s.cancel = cancel
        s.emitter = emitter
        s.done = make(chan struct{})
        s.crawlerMu.Unlock()

        // Subscribe the SSE broadcaster to crawler events.
        eventsCh := emitter.Subscribe(256)
        go s.broadcastPump(eventsCh)

        // Build the list of seed URLs. The primary `URL` field is always the
        // first seed; any additional URLs in `URLs` are appended. We dedupe
        // to avoid enqueueing the same URL twice.
        seeds := []string{req.URL}
        for _, u := range req.URLs {
                u = strings.TrimSpace(u)
                if u == "" {
                        continue
                }
                seeds = append(seeds, u)
        }
        seeds = dedupStrings(seeds)

        // Run the crawler in a goroutine. If we have a single seed we use the
        // regular Run path; otherwise we use RunSeeds which shares the
        // visited-set across every seed.
        go func() {
                defer func() {
                        s.crawlerMu.Lock()
                        s.crawler = nil
                        s.cancel = nil
                        s.crawlerMu.Unlock()
                        close(s.done)
                }()
                var err error
                if len(seeds) == 1 {
                        _, err = c.Run(ctx, seeds[0])
                } else {
                        _, err = c.RunSeeds(ctx, seeds)
                }
                if err != nil {
                        fmt.Printf("crawl error: %v\n", err)
                }
        }()

        writeJSON(w, 200, map[string]any{
                "status": "started",
                "seeds":  seeds,
                "batch":  len(seeds) > 1,
        })
}

// handleJobsStart is the multi-job queue entry point. The GUI collects a
// list of fully-configured jobs (one per site, each with its own output
// dir and crawler options) and submits them here. The server runs them
// sequentially in a single goroutine; /api/pause, /api/resume and
// /api/stop control the CURRENT job (stop also aborts the remaining queue).
//
// Each job gets its own Crawler instance with its own visited-set, so URLs
// fetched for job #1 are still re-fetched for job #2 if they overlap. This
// is intentional: the user wants a complete mirror per site, not a global
// deduplication across sites.
func (s *Server) handleJobsStart(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        var req struct {
                Jobs []Job `json:"jobs"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeJSON(w, 400, map[string]string{"error": err.Error()})
                return
        }
        if len(req.Jobs) == 0 {
                writeJSON(w, 400, map[string]string{"error": "no jobs provided"})
                return
        }
        // Validate every job up front so we fail fast on bad input rather
        // than midway through a long queue.
        for i, j := range req.Jobs {
                if j.URL == "" {
                        writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("job %d: url is required", i+1)})
                        return
                }
                if j.OutputDir == "" {
                        writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("job %d: output_dir is required", i+1)})
                        return
                }
        }

        // Make sure no crawl is already running
        s.crawlerMu.Lock()
        if s.crawler != nil {
                s.crawlerMu.Unlock()
                writeJSON(w, 409, map[string]string{"error": "crawl already running"})
                return
        }
        s.crawlerMu.Unlock()

        // Install the queue state. jobIndex starts at -1 and is bumped to 0
        // by runJobs before the first job starts.
        s.jobQueueMu.Lock()
        s.jobs = req.Jobs
        s.jobIndex = -1
        s.jobState = "running"
        s.jobQueueMu.Unlock()

        // Single shared emitter for the whole queue — every subscriber sees
        // events from every job, tagged with job_index/job_total/job_url so
        // the UI can highlight the active job in the list.
        emitter := crawler.NewEventEmitter()
        eventsCh := emitter.Subscribe(256)
        go s.broadcastPump(eventsCh)

        ctx, cancel := context.WithCancel(context.Background())
        s.crawlerMu.Lock()
        s.emitter = emitter
        s.cancel = cancel
        s.done = make(chan struct{})
        s.crawlerMu.Unlock()

        go func() {
                defer func() {
                        s.crawlerMu.Lock()
                        s.crawler = nil
                        s.cancel = nil
                        s.emitter = nil
                        s.crawlerMu.Unlock()
                        close(s.done)
                        emitter.Close()
                        s.jobQueueMu.Lock()
                        if s.jobState != "stopped" {
                                s.jobState = "done"
                        }
                        s.jobQueueMu.Unlock()
                }()
                s.runJobs(ctx, req.Jobs, emitter)
        }()

        writeJSON(w, 200, map[string]any{
                "status": "started",
                "jobs":   len(req.Jobs),
        })
}

// runJobs runs every job in the queue sequentially. If ctx is cancelled
// (e.g. the user clicked Stop), the current job is aborted and remaining
// jobs are skipped. Pause/Resume applies to the currently running job
// only; the queue advances to the next job as soon as the current one
// finishes (either cleanly or with errors).
func (s *Server) runJobs(ctx context.Context, jobs []Job, emitter *crawler.EventEmitter) {
        total := len(jobs)
        for i, job := range jobs {
                if ctx.Err() != nil {
                        return
                }
                s.jobQueueMu.Lock()
                s.jobIndex = i
                s.jobState = "running"
                s.jobQueueMu.Unlock()

                emitter.Emit(crawler.ProgressEvent{
                        Type:     crawler.EventLog,
                        Time:     time.Now(),
                        JobIndex: i,
                        JobTotal: total,
                        JobURL:   job.URL,
                        Msg:      fmt.Sprintf("starting job %d/%d: %s", i+1, total, job.URL),
                })

                cfg := jobToConfig(job)
                cfg.Events = emitter
                cfg.JobIndex = i
                cfg.JobTotal = total
                cfg.JobURL = job.URL
                cfg.Verbose = false

                c, err := crawler.New(cfg)
                if err != nil {
                        emitter.Emit(crawler.ProgressEvent{
                                Type:     crawler.EventLog,
                                Time:     time.Now(),
                                JobIndex: i,
                                JobTotal: total,
                                JobURL:   job.URL,
                                Msg:      fmt.Sprintf("job %d failed to start: %v", i+1, err),
                        })
                        continue
                }

                s.crawlerMu.Lock()
                s.crawler = c
                s.crawlerMu.Unlock()

                if _, err := c.Run(ctx, job.URL); err != nil {
                        // Run already emits an end event; we just log and move on.
                        fmt.Printf("job %d (%s) error: %v\n", i+1, job.URL, err)
                }
        }
}

// jobToConfig converts a Job (the JSON payload from the GUI) into a
// crawler.Config. Cookie-string parsing is shared with the legacy
// /api/start path; we keep it inline here for clarity.
func jobToConfig(j Job) crawler.Config {
        cfg := crawler.Config{
                OutputDir:               j.OutputDir,
                Workers:                 j.Workers,
                MaxURLs:                 j.MaxURLs,
                MaxDepth:                j.MaxDepth,
                SameDomainOnly:          j.SameDomain,
                AllowExternalSubdomains: j.AllowSubs,
                SkipAssets:              j.SkipAssets,
                InsecureTLS:             j.InsecureTLS,
                SaveManifest:            j.Manifest,
                Proxy:                   j.Proxy,
                UserAgent:               j.UserAgent,
                Timeout:                 time.Duration(j.Timeout) * time.Second,
        }
        if j.AllowedHosts != "" {
                cfg.AllowedHosts = strings.Split(j.AllowedHosts, ",")
                for i := range cfg.AllowedHosts {
                        cfg.AllowedHosts[i] = strings.TrimSpace(cfg.AllowedHosts[i])
                }
        }
        if j.AssetExt != "" {
                cfg.AssetExtensions = strings.Split(j.AssetExt, ",")
                for i := range cfg.AssetExtensions {
                        cfg.AssetExtensions[i] = strings.TrimSpace(cfg.AssetExtensions[i])
                }
        }
        if j.Cookies != "" {
                if jar, err := cookiejar.New(nil); err == nil {
                        for _, kv := range strings.Split(j.Cookies, ";") {
                                kv = strings.TrimSpace(kv)
                                if kv == "" {
                                        continue
                                }
                                parts := strings.SplitN(kv, "=", 2)
                                c := &http.Cookie{}
                                if len(parts) == 2 {
                                        c.Name = strings.TrimSpace(parts[0])
                                        c.Value = strings.TrimSpace(parts[1])
                                } else {
                                        c.Name = strings.TrimSpace(parts[0])
                                }
                                if c.Name != "" {
                                        if u, err := url.Parse(j.URL); err == nil {
                                                jar.SetCookies(&url.URL{Scheme: u.Scheme, Host: u.Host}, []*http.Cookie{c})
                                        }
                                }
                        }
                        cfg.CookieJar = jar
                }
        }
        return cfg
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        s.crawlerMu.Lock()
        cancel := s.cancel
        s.crawlerMu.Unlock()
        if cancel == nil {
                writeJSON(w, 400, map[string]string{"error": "no crawl running"})
                return
        }
        cancel()
        // Mark the queue as stopped so /api/status reflects it immediately
        // rather than waiting for the goroutine to drain.
        s.jobQueueMu.Lock()
        s.jobState = "stopped"
        s.jobQueueMu.Unlock()
        writeJSON(w, 200, map[string]string{"status": "stopped"})
}

// handlePause suspends the running crawl. Workers finish any in-flight fetch
// before pausing, so the crawl is always in a consistent state. Calling
// pause when no crawl is running, or when already paused, is a no-op.
func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        s.crawlerMu.Lock()
        c := s.crawler
        s.crawlerMu.Unlock()
        if c == nil {
                writeJSON(w, 400, map[string]string{"error": "no crawl running"})
                return
        }
        c.Pause()
        s.jobQueueMu.Lock()
        s.jobState = "paused"
        s.jobQueueMu.Unlock()
        writeJSON(w, 200, map[string]string{"status": "paused"})
}

// handleResume unblocks a previously-paused crawl.
func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        s.crawlerMu.Lock()
        c := s.crawler
        s.crawlerMu.Unlock()
        if c == nil {
                writeJSON(w, 400, map[string]string{"error": "no crawl running"})
                return
        }
        c.Resume()
        s.jobQueueMu.Lock()
        s.jobState = "running"
        s.jobQueueMu.Unlock()
        writeJSON(w, 200, map[string]string{"status": "resumed"})
}

// handleStatus returns the current crawler state: idle / running / paused,
// plus a snapshot of the counters AND the job-queue state (which job is
// currently running, how many jobs total, etc.). The GUI polls this on
// load to restore its UI after a page refresh.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                http.Error(w, "method not allowed", 405)
                return
        }
        s.crawlerMu.Lock()
        c := s.crawler
        s.crawlerMu.Unlock()
        s.jobQueueMu.RLock()
        queueState := s.jobState
        queueIndex := s.jobIndex
        queueTotal := len(s.jobs)
        var queueURL string
        if queueIndex >= 0 && queueIndex < queueTotal {
                queueURL = s.jobs[queueIndex].URL
        }
        s.jobQueueMu.RUnlock()

        if c == nil {
                writeJSON(w, 200, map[string]any{
                        "state":      "idle",
                        "queue_state": queueState,
                })
                return
        }
        state := "running"
        if c.IsPaused() {
                state = "paused"
        }
        visited, downloaded, failed, skipped, bytes, pages, assets := c.SnapshotCounters()
        writeJSON(w, 200, map[string]any{
                "state":      state,
                "queue_state": queueState,
                "queue_index": queueIndex, // 0-based, -1 means "not started yet"
                "queue_total": queueTotal,
                "queue_url":   queueURL,
                "visited":     visited,
                "downloaded":  downloaded,
                "failed":      failed,
                "skipped":     skipped,
                "bytes":       bytes,
                "pages":       pages,
                "assets":      assets,
        })
}

// handleEvents is the SSE endpoint. It registers a subscriber channel,
// streams events to the client, and unregisters when the client disconnects.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
        flusher, ok := w.(http.Flusher)
        if !ok {
                http.Error(w, "streaming not supported", 500)
                return
        }
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        w.Header().Set("Access-Control-Allow-Origin", "*")

        ch := make(chan crawler.ProgressEvent, 64)
        s.subsMu.Lock()
        s.subs[ch] = struct{}{}
        s.subsMu.Unlock()

        defer func() {
                s.subsMu.Lock()
                delete(s.subs, ch)
                s.subsMu.Unlock()
                close(ch)
        }()

        // Send an initial ping so the client knows the connection is alive.
        fmt.Fprintf(w, ": ping\n\n")
        flusher.Flush()

        for {
                select {
                case <-r.Context().Done():
                        return
                case ev, ok := <-ch:
                        if !ok {
                                return
                        }
                        data, _ := json.Marshal(ev)
                        fmt.Fprintf(w, "data: %s\n\n", data)
                        flusher.Flush()
                }
        }
}

// broadcastPump reads events from the crawler's emitter and forwards them
// to every SSE subscriber.
func (s *Server) broadcastPump(ch chan crawler.ProgressEvent) {
        for ev := range ch {
                s.subsMu.RLock()
                for sub := range s.subs {
                        select {
                        case sub <- ev:
                        default:
                                // drop if subscriber is slow
                        }
                }
                s.subsMu.RUnlock()
        }
}

// handleOpen opens the given path in the OS file manager.
func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        var req struct {
                Path string `json:"path"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeJSON(w, 400, map[string]string{"error": err.Error()})
                return
        }
        if req.Path == "" {
                writeJSON(w, 400, map[string]string{"error": "path is required"})
                return
        }
        openInOS(req.Path)
        writeJSON(w, 200, map[string]string{"status": "ok"})
}

// handleServe opens the mirrored index.html in the browser.
func (s *Server) handleServe(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }
        var req struct {
                Path string `json:"path"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeJSON(w, 400, map[string]string{"error": err.Error()})
                return
        }
        abs, _ := filepath.Abs(req.Path)
        indexPath := filepath.Join(abs, "index.html")
        if _, err := os.Stat(indexPath); err == nil {
                openInOS(indexPath)
                writeJSON(w, 200, map[string]string{"url": "file:///" + filepath.ToSlash(indexPath)})
        } else {
                // Fall back to opening the directory
                openInOS(abs)
                writeJSON(w, 200, map[string]string{"url": "file:///" + filepath.ToSlash(abs)})
        }
}

// --- Helpers ---

// handleBrowse opens a NATIVE folder-picker dialog on the user's OS and
// returns the selected path. This is what powers the "Browse..." button in
// the GUI — instead of asking the user to type a path into a text prompt,
// we shell out to the platform's native dialog:
//
//   - Windows: PowerShell + System.Windows.Forms.FolderBrowserDialog
//   - macOS:   osascript "choose folder"
//   - Linux:   zenity --file-selection --directory (falls back to kdialog)
//
// The handler blocks until the user picks a folder or cancels. We give it
// a generous 5-minute timeout on the HTTP side so the user has time to
// navigate. If the user cancels, we return path="" and the frontend leaves
// the input unchanged.
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "method not allowed", 405)
                return
        }

        // Optional: accept a "current" path to start the dialog in. Helps the
        // user by opening the dialog at the directory they're already looking
        // at rather than the OS default.
        var req struct {
                Current string `json:"current"`
        }
        // Body is optional; ignore decode errors.
        _ = json.NewDecoder(r.Body).Decode(&req)

        path, err := browseDirectory(req.Current)
        if err != nil {
                // The user cancelling is NOT an error from our perspective, but
                // some tools report it as a non-zero exit code. We check for an
                // empty path below to detect cancellation.
                // Real errors (e.g. zenity not installed) get reported to the UI
                // so the user knows why the dialog didn't open.
                if path != "" {
                        writeJSON(w, 200, map[string]string{"path": path})
                        return
                }
                writeJSON(w, 200, map[string]string{
                        "path":  "",
                        "error": err.Error(),
                })
                return
        }
        writeJSON(w, 200, map[string]string{"path": path})
}

// browseDirectory runs the platform-native folder picker and returns the
// selected path. Returns ("", nil) when the user cancels. Returns ("", err)
// when no picker tool is available (e.g. headless Linux without zenity).
func browseDirectory(startDir string) (string, error) {
        switch runtime.GOOS {
        case "windows":
                return browseWindows(startDir)
        case "darwin":
                return browseMacOS(startDir)
        default:
                return browseLinux(startDir)
        }
}

// browseWindows uses PowerShell to show the .NET FolderBrowserDialog.
// We use -NoProfile for fast startup and -OutputFormat Text so the
// selected path comes back on stdout as plain text.
func browseWindows(startDir string) (string, error) {
        // Build the PowerShell script. SelectedPath is pre-set to startDir so
        // the dialog opens where the user left off (best-effort — the .NET
        // dialog only honors this if the path exists).
        ps := `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.FolderBrowserDialog
$d.Description = 'Select output folder'`
        if startDir != "" {
                ps += "\n$d.SelectedPath = '" + strings.ReplaceAll(startDir, "'", "''") + "'"
        }
        ps += `
if ($d.ShowDialog() -eq 'OK') { Write-Output $d.SelectedPath }`
        cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-OutputFormat", "Text", "-Command", ps)
        out, err := cmd.Output()
        if err != nil {
                return "", fmt.Errorf("powershell folder dialog: %w", err)
        }
        p := strings.TrimSpace(string(out))
        return p, nil
}

// browseMacOS uses osascript to invoke the AppleScript "choose folder"
// command. The POSIX path of the chosen folder is returned on stdout.
func browseMacOS(startDir string) (string, error) {
        script := `tell application (path to frontmost application as text) to set chosenFolder to choose folder
return POSIX path of chosenFolder`
        cmd := exec.Command("osascript", "-e", script)
        out, err := cmd.Output()
        if err != nil {
                // osascript exits non-zero when the user clicks Cancel. We treat
                // that as "no selection" rather than an error.
                return "", nil
        }
        p := strings.TrimSpace(string(out))
        return p, nil
}

// browseLinux tries zenity first (GNOME / most distros), then kdialog
// (KDE). Both print the selected path to stdout and exit non-zero on
// cancel.
func browseLinux(startDir string) (string, error) {
        // Try zenity.
        if _, err := exec.LookPath("zenity"); err == nil {
                args := []string{"--file-selection", "--directory", "--title=Select output folder"}
                if startDir != "" {
                        args = append(args, "--filename="+startDir+"/")
                }
                cmd := exec.Command("zenity", args...)
                out, err := cmd.Output()
                if err == nil {
                        return strings.TrimSpace(string(out)), nil
                }
                // Non-zero exit usually means cancel; return empty path silently.
                return "", nil
        }
        // Fall back to kdialog.
        if _, err := exec.LookPath("kdialog"); err == nil {
                args := []string{"--getexistingdirectory"}
                if startDir != "" {
                        args = append(args, startDir)
                } else {
                        args = append(args, ".")
                }
                cmd := exec.Command("kdialog", args...)
                out, err := cmd.Output()
                if err == nil {
                        return strings.TrimSpace(string(out)), nil
                }
                return "", nil
        }
        return "", fmt.Errorf("no folder picker available (install zenity or kdialog)")
}

// dedupStrings returns the input with duplicates removed, preserving order.
// Used to dedupe the seed URL list before passing it to the crawler.
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

func writeJSON(w http.ResponseWriter, status int, v any) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(v)
}

func openInOS(target string) {
        var cmd *exec.Cmd
        switch runtime.GOOS {
        case "windows":
                cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
        case "darwin":
                cmd = exec.Command("open", target)
        default:
                cmd = exec.Command("xdg-open", target)
        }
        _ = cmd.Start()
}
