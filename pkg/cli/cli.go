// Package cli wires the cobra command flags to the crawler configuration.
package cli

import (
        "context"
        "fmt"
        "net/http"
        "net/http/cookiejar"
        "net/url"
        "os"
        "os/signal"
        "path/filepath"
        "strings"
        "syscall"
        "time"

        "github.com/spf13/cobra"

        "github.com/a-talebifard/webclone/pkg/crawler"
        "github.com/a-talebifard/webclone/pkg/urlx"
)

// flag values populated by cobra; passed into the crawler Config.
var (
        flagOutput       string
        flagWorkers      int
        flagMaxURLs      int
        flagMaxDepth     int
        flagSameDomain   bool
        flagAllowSubs    bool
        flagAllowedHosts []string
        flagSkipAssets   bool
        flagAssetExt     []string
        flagTimeout      time.Duration
        flagUserAgent    string
        flagProxy        string
        flagInsecureTLS  bool
        flagVerbose      bool
        flagServe        bool
        flagServePort    int
        flagManifest     bool
        flagCookies      []string
        flagQuiet        bool
)

// rootCmd is the entry-point command for webclone.
var rootCmd = &cobra.Command{
        Use:   "webclone <url> [url2 ...]",
        Short: "Mirror entire websites to disk",
        Long: `webclone downloads a website (or several) to your local disk,
preserving the URL hierarchy so the site can be browsed offline.

Unlike a single-page cloner, webclone follows internal links recursively
and saves every page and asset (HTML, CSS, JS, images, fonts, ...) it finds,
mirroring the site's URL structure under the output directory.

Examples:
  # Mirror a single site, all internal pages, default settings:
  webclone https://example.com

  # Mirror only same-domain pages (skip external links):
  webclone --same-domain https://example.com

  # Mirror to a custom output dir with a 50k URL cap:
  webclone -o ./mirror --max-urls 50000 https://example.com

  # Limit to 3 link hops from the seed:
  webclone --max-depth 3 https://example.com

  # Explicit host allowlist (crawls example.com AND cdn.example.com):
  webclone --allowed-hosts example.com,cdn.example.com https://example.com

  # Use a custom user-agent and proxy:
  webclone -u "Mozilla/5.0 ..." --proxy http://localhost:8080 https://example.com
`,
        Args: cobra.MinimumNArgs(1),
        RunE: runRoot,
}

// Execute runs the root command.
func Execute() {
        if err := rootCmd.Execute(); err != nil {
                os.Exit(1)
        }
}

func init() {
        flags := rootCmd.Flags()
        flags.SortFlags = false

        flags.StringVarP(&flagOutput, "output", "o", "", "Output directory for the mirror tree (default: current directory)")
        flags.IntVarP(&flagWorkers, "workers", "w", 5, "Number of concurrent fetch workers")
        flags.IntVar(&flagMaxURLs, "max-urls", 10000, "Hard cap on number of URLs to fetch (0 = unlimited)")
        flags.IntVar(&flagMaxDepth, "max-depth", 0, "Max link-hops from the seed URL (0 = unlimited)")
        flags.BoolVar(&flagSameDomain, "same-domain", true, "Only crawl URLs on the seed's registered domain (default: true)")
        flags.BoolVar(&flagAllowSubs, "allow-subdomains", true, "When --same-domain is set, also follow subdomains of the seed host")
        flags.StringSliceVar(&flagAllowedHosts, "allowed-hosts", nil, "Comma-separated explicit list of hostnames to crawl (overrides --same-domain)")
        flags.BoolVar(&flagSkipAssets, "skip-assets", false, "Only download HTML pages, skip CSS/JS/images")
        flags.StringSliceVar(&flagAssetExt, "asset-ext", nil, "Comma-separated list of asset extensions to download (e.g. .css,.js,.png). Empty = all")
        flags.DurationVar(&flagTimeout, "timeout", 60*time.Second, "Per-request timeout")
        flags.StringVarP(&flagUserAgent, "user-agent", "u", "", "Custom User-Agent string")
        flags.StringVarP(&flagProxy, "proxy", "p", "", "Proxy URL (http://, https://, socks5://)")
        flags.BoolVar(&flagInsecureTLS, "insecure-tls", false, "Skip TLS certificate verification")
        flags.BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose per-URL log output")
        flags.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress all non-error output")
        flags.BoolVarP(&flagServe, "serve", "s", false, "After mirroring, serve the output dir on a local HTTP server")
        flags.IntVarP(&flagServePort, "serve-port", "P", 8080, "Port for --serve")
        flags.BoolVar(&flagManifest, "manifest", true, "Write manifest.json with all crawled URLs")
        flags.StringSliceVarP(&flagCookies, "cookie", "C", nil, "Pre-set cookies (name=value; ...) applied to every crawl host")
}

func runRoot(cmd *cobra.Command, args []string) error {
        if flagQuiet {
                flagVerbose = false
        }

        // Validate and normalize seed URLs.
        var seeds []string
        for _, a := range args {
                if !urlx.IsCrawlable(a) {
                        return fmt.Errorf("invalid URL %q", a)
                }
                canonical, err := urlx.Canonicalize(a, a)
                if err != nil {
                        return fmt.Errorf("invalid URL %q: %w", a, err)
                }
                seeds = append(seeds, canonical)
        }

        outputDir := flagOutput
        if outputDir == "" {
                wd, err := os.Getwd()
                if err != nil {
                        return err
                }
                outputDir = wd
        }

        // Shared cookie jar (optional)
        var jar *cookiejar.Jar
        if len(flagCookies) > 0 {
                j, err := cookiejar.New(nil)
                if err != nil {
                        return fmt.Errorf("cookie jar: %w", err)
                }
                cookies := parseCookies(flagCookies)
                for _, seed := range seeds {
                        j.SetCookies(hostOnlyURL(seed), cookies)
                }
                jar = j
        }

        // Signal handling - cancel crawl on Ctrl+C / SIGTERM
        ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
        defer stop()

        for _, seed := range seeds {
                cfg := crawler.Config{
                        OutputDir:               outputDir,
                        Workers:                 flagWorkers,
                        MaxURLs:                 flagMaxURLs,
                        MaxDepth:                flagMaxDepth,
                        SameDomainOnly:          flagSameDomain,
                        AllowExternalSubdomains: flagAllowSubs,
                        AllowedHosts:            flagAllowedHosts,
                        AssetExtensions:         flagAssetExt,
                        SkipAssets:              flagSkipAssets,
                        Timeout:                 flagTimeout,
                        UserAgent:               flagUserAgent,
                        Proxy:                   flagProxy,
                        InsecureTLS:             flagInsecureTLS,
                        CookieJar:               jar,
                        Verbose:                 flagVerbose,
                        SaveManifest:            flagManifest,
                }

                c, err := crawler.New(cfg)
                if err != nil {
                        return fmt.Errorf("init crawler for %s: %w", seed, err)
                }
                if _, err := c.Run(ctx, seed); err != nil {
                        return fmt.Errorf("crawl %s: %w", seed, err)
                }
        }

        if flagServe {
                return serveOutput(outputDir, flagServePort)
        }
        return nil
}

// parseCookies parses "name=value; name2=value2" cookie strings into a list
// of http.Cookie values that will be applied to every seed host.
func parseCookies(raw []string) []*http.Cookie {
        var out []*http.Cookie
        for _, s := range raw {
                for _, kv := range strings.Split(s, ";") {
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
                                out = append(out, c)
                        }
                }
        }
        return out
}

// hostOnlyURL converts an absolute URL string to a *url.URL that contains
// only the scheme + host info (no path/query). cookiejar.SetCookies keys
// cookies by host so this is what it expects.
func hostOnlyURL(rawURL string) *url.URL {
        u, err := url.Parse(rawURL)
        if err != nil {
                return &url.URL{Scheme: "http"}
        }
        return &url.URL{Scheme: u.Scheme, Host: u.Host}
}

// serveOutput starts a static file server on the given port serving the
// mirror directory. It blocks until interrupted.
func serveOutput(dir string, port int) error {
        abs, err := filepath.Abs(dir)
        if err != nil {
                return err
        }
        fmt.Printf("webclone: serving %s at http://localhost:%d/\n", abs, port)
        srv := &http.Server{
                Addr:              fmt.Sprintf(":%d", port),
                Handler:           http.FileServer(http.Dir(abs)),
                ReadHeaderTimeout: 30 * time.Second,
        }
        return srv.ListenAndServe()
}
