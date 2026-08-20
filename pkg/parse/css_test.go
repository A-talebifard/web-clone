package parse

import (
	"testing"
)

func TestRewriteCSSURLs(t *testing.T) {
	// pageURL is https://example.com/page.html, so pageDir is "example.com"
	// (the root of the example.com mirror). All rewritten URLs that live at
	// the example.com root will be relative paths like "img/x.png" with no
	// leading "../".
	body := `
        .a { background: url("/img/bg.png"); }
        .b { background: url('/img/bg2.png'); }
        .c { background: url(/img/bg3.png); }
        @import "/css/sub.css";
        @import '/css/sub2.css';
        @import url("/css/sub3.css");
        .d { background: url("data:,"); }
`
	shouldMirror := func(s string) bool { return true }
	rewritten, urls := RewriteCSS(body, "https://example.com/page.html", "example.com", shouldMirror)
	// We expect: 3 url() refs (in .a, .b, .c) + 2 @import "..." refs +
	// 1 @import url("...") (caught by the url() regex). Total: 6 urls.
	if len(urls) != 6 {
		t.Errorf("expected 6 urls; got %d: %v", len(urls), urls)
	}
	wantSubs := []string{
		`url("img/bg.png")`,
		`url('img/bg2.png')`,
		`url(img/bg3.png)`,
		`@import "css/sub.css"`,
		`@import 'css/sub2.css'`,
	}
	for _, w := range wantSubs {
		if !contains(rewritten, w) {
			t.Errorf("rewritten CSS missing %q\ngot: %s", w, rewritten)
		}
	}
}

func TestRewriteCSSExternal(t *testing.T) {
	body := `.a { background: url("https://cdn.example.com/x.png"); }`
	shouldMirror := func(s string) bool { return false } // don't mirror external
	rewritten, urls := RewriteCSS(body, "https://example.com/page.html", "example.com", shouldMirror)
	if len(urls) != 0 {
		t.Errorf("expected 0 urls when not mirroring external; got %d", len(urls))
	}
	if !contains(rewritten, "https://cdn.example.com/x.png") {
		t.Errorf("expected URL left intact when not mirroring; got: %s", rewritten)
	}
}

func TestRewriteCSSNestedPage(t *testing.T) {
	// pageURL is https://example.com/blog/post.html, so pageDir is
	// "example.com/blog". URLs at example.com/img/ require going up one dir.
	body := `.a { background: url("/img/bg.png"); }`
	shouldMirror := func(s string) bool { return true }
	rewritten, urls := RewriteCSS(body, "https://example.com/blog/post.html", "example.com/blog", shouldMirror)
	if len(urls) != 1 {
		t.Errorf("expected 1 url; got %d: %v", len(urls), urls)
	}
	if !contains(rewritten, `url("../img/bg.png")`) {
		t.Errorf("expected ../img/bg.png; got: %s", rewritten)
	}
}

func TestParseCSSURLs(t *testing.T) {
	body := `
                .a { background: url("/img/a.png"); }
                .b { background: url('/img/b.png'); }
                .c { background: url(/img/c.png); }
                @import "/css/x.css";
                @import '/css/y.css';
                .d { background: url("data:,"); }
        `
	urls := ParseCSSURLs(body, "https://example.com/page.html")
	if len(urls) != 5 {
		t.Errorf("expected 5 urls; got %d: %v", len(urls), urls)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
