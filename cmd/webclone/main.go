// Command webclone is a recursive website mirror tool.
//
// It downloads an entire website (HTML pages plus all referenced assets:
// CSS, JS, images, fonts, ...) to a local directory, preserving the URL
// structure so that the mirrored copy can be browsed offline.
//
// Usage examples:
//
//	webclone https://example.com
//	webclone -o ./mirror --max-depth 3 https://example.com
//	webclone --same-domain --workers 10 https://example.com
package main

import "github.com/a-talebifard/webclone/pkg/cli"

func main() {
	cli.Execute()
}
