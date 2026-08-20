package urlx

import "testing"

func TestMirrorPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"root", "https://example.com/", "example.com/index.html"},
		{"bare host", "https://example.com", "example.com/index.html"},
		{"path no ext", "https://example.com/foo", "example.com/foo/index.html"},
		{"path trailing slash", "https://example.com/foo/", "example.com/foo/index.html"},
		{"nested path no ext", "https://example.com/foo/bar", "example.com/foo/bar/index.html"},
		{"path with ext", "https://example.com/foo/bar.html", "example.com/foo/bar.html"},
		{"css asset", "https://example.com/style.css", "example.com/style.css"},
		{"nested asset", "https://example.com/css/main.css", "example.com/css/main.css"},
		{"external", "https://other.com/page", "other.com/page/index.html"},
		{"uppercase host", "https://Example.com/Foo", "example.com/Foo/index.html"},
		{"default port http", "http://example.com:80/foo", "example.com/foo/index.html"},
		{"default port https", "https://example.com:443/foo", "example.com/foo/index.html"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := MirrorPath(c.in)
			if err != nil {
				t.Fatalf("MirrorPath(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("MirrorPath(%q) = %q; want %q", c.in, got, c.want)
			}
		})
	}
}

func TestMirrorPathWithQuery(t *testing.T) {
	// Query produces a hash suffix; we just check it's present and not the
	// plain filename.
	got, err := MirrorPath("https://example.com/style.css?v=123")
	if err != nil {
		t.Fatal(err)
	}
	if got == "example.com/style.css" {
		t.Errorf("expected query hash suffix; got %q", got)
	}
	if !contains(got, "style_") || !contains(got, ".css") {
		t.Errorf("expected style_<hash>.css pattern; got %q", got)
	}
}

func TestRelativePath(t *testing.T) {
	cases := []struct {
		from, to, want string
	}{
		// fromDir is the directory the current page lives in.
		// "example.com" represents the root dir of the example.com mirror.
		{"example.com", "example.com/about.html", "about.html"},
		{"example.com", "example.com/foo/index.html", "foo/index.html"},
		// "example.com/foo" represents the directory example.com/foo/.
		// Going from there to example.com/about.html requires one ".." to
		// leave the foo/ dir and reach example.com/.
		{"example.com/foo", "example.com/about.html", "../about.html"},
		{"example.com/foo", "example.com/foo/bar.html", "bar.html"},
		{"example.com/foo/bar", "example.com/foo/bar.html", "../bar.html"},
		// Cross-host: from example.com/foo/ to other.com/page/index.html
		// requires going up foo/ then up example.com/ to reach the root,
		// then down into other.com/page/.
		{"example.com/foo", "other.com/page/index.html", "../../other.com/page/index.html"},
	}
	for _, c := range cases {
		got := RelativePath(c.from, c.to)
		if got != c.want {
			t.Errorf("RelativePath(%q, %q) = %q; want %q", c.from, c.to, got, c.want)
		}
	}
}

func TestCanonicalize(t *testing.T) {
	cases := []struct {
		raw, base, want string
	}{
		{"/about", "https://example.com/", "https://example.com/about"},
		// Relative "about.html" against base "/foo" (file) resolves to "/about.html"
		// because /foo's directory is just "/".
		{"about.html", "https://example.com/foo", "https://example.com/about.html"},
		{"about.html", "https://example.com/foo/", "https://example.com/foo/about.html"},
		{"//other.com/x", "https://example.com/", "https://other.com/x"},
		{"https://example.com/x#frag", "https://example.com/", "https://example.com/x"},
		// "../foo" against /a/b: /a/b's dir is /a/, then .. -> /, then foo -> /foo
		{"../foo", "https://example.com/a/b", "https://example.com/foo"},
		{"./foo", "https://example.com/a/b", "https://example.com/a/foo"},
	}
	for _, c := range cases {
		got, err := Canonicalize(c.raw, c.base)
		if err != nil {
			t.Fatalf("Canonicalize(%q, %q): %v", c.raw, c.base, err)
		}
		if got != c.want {
			t.Errorf("Canonicalize(%q, %q) = %q; want %q", c.raw, c.base, got, c.want)
		}
	}
}

func TestIsCrawlable(t *testing.T) {
	good := []string{"http://x.com/", "https://x.com", "/foo", "foo.html", "?x=1"}
	bad := []string{"mailto:a@b.com", "tel:+1234", "javascript:alert(1)", "data:,", "blob:xx", "#anchor", "ftp://x.com", "file:///x", ""}
	for _, s := range good {
		if !IsCrawlable(s) {
			t.Errorf("IsCrawlable(%q) = false; want true", s)
		}
	}
	for _, s := range bad {
		if IsCrawlable(s) {
			t.Errorf("IsCrawlable(%q) = true; want false", s)
		}
	}
}

func TestIsHTMLAsset(t *testing.T) {
	html := []string{
		"https://example.com/",
		"https://example.com/foo",
		"https://example.com/foo.html",
		"https://example.com/foo.php",
	}
	nonHTML := []string{
		"https://example.com/style.css",
		"https://example.com/app.js",
		"https://example.com/img.png",
		"https://example.com/img.jpg",
	}
	for _, u := range html {
		if !IsHTMLAsset(u) {
			t.Errorf("IsHTMLAsset(%q) = false; want true", u)
		}
	}
	for _, u := range nonHTML {
		if IsHTMLAsset(u) {
			t.Errorf("IsHTMLAsset(%q) = true; want false", u)
		}
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
