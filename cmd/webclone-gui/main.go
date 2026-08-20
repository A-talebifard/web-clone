// Command webclone-gui launches a browser-based GUI for webclone.
//
// The GUI is served by a local HTTP server (default port 8080) and opens
// automatically in the user's default browser. The interface is bilingual
// (Persian/English) with a dark modern theme and the Vazirmatn font.
//
// Usage:
//
//	webclone-gui [port]
//
// Default port is 8080.
//
// This command has zero CGO dependencies and works on Windows/macOS/Linux
// without any external libraries or compilers.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/a-talebifard/webclone/pkg/webui"
)

func main() {
	port := 8080
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}

	srv := webui.New(port)
	srv.OpenBrowser()
	fmt.Printf("webclone GUI: http://127.0.0.1:%d/\n", port)
	fmt.Println("Press Ctrl+C to stop.")
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
