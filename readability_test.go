package readability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type expectedMetadata struct {
	Title         string  `json:"title"`
	Byline        *string `json:"byline"`
	Dir           *string `json:"dir"`
	Lang          *string `json:"lang"`
	Excerpt       *string `json:"excerpt"`
	SiteName      *string `json:"siteName"`
	PublishedTime *string `json:"publishedTime"`
	Readerable    bool    `json:"readerable"`
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func TestReadabilityFixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata/test-pages")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "test-pages", name)

			sourceBytes, err := os.ReadFile(filepath.Join(dir, "source.html"))
			if err != nil {
				t.Fatal(err)
			}

			expectedHTMLBytes, err := os.ReadFile(filepath.Join(dir, "expected.html"))
			if err != nil {
				t.Fatal(err)
			}

			metaBytes, err := os.ReadFile(filepath.Join(dir, "expected-metadata.json"))
			if err != nil {
				t.Fatal(err)
			}

			var meta expectedMetadata
			if err := json.Unmarshal(metaBytes, &meta); err != nil {
				t.Fatal(err)
			}

			article, err := Parse(strings.NewReader(string(sourceBytes)), "http://fakehost/test/page.html")
			if err != nil {
				t.Fatal(err)
			}

			if article == nil {
				t.Fatal("article is nil, expected content")
			}

			if article.Title != meta.Title {
				t.Errorf("title mismatch:\n  got:  %q\n  want: %q", article.Title, meta.Title)
			}

			if article.Byline != strVal(meta.Byline) {
				t.Errorf("byline mismatch:\n  got:  %q\n  want: %q", article.Byline, strVal(meta.Byline))
			}

			if article.Dir != strVal(meta.Dir) {
				t.Errorf("dir mismatch:\n  got:  %q\n  want: %q", article.Dir, strVal(meta.Dir))
			}

			if article.Excerpt != strVal(meta.Excerpt) {
				t.Errorf("excerpt mismatch:\n  got:  %q\n  want: %q", article.Excerpt, strVal(meta.Excerpt))
			}

			if article.SiteName != strVal(meta.SiteName) {
				t.Errorf("siteName mismatch:\n  got:  %q\n  want: %q", article.SiteName, strVal(meta.SiteName))
			}

			if article.PublishedTime != strVal(meta.PublishedTime) {
				t.Errorf("publishedTime mismatch:\n  got:  %q\n  want: %q", article.PublishedTime, strVal(meta.PublishedTime))
			}

			expectedHTML := strings.TrimSpace(string(expectedHTMLBytes))
			gotHTML := strings.TrimSpace(article.Content)

			if normalizeHTML(gotHTML) != normalizeHTML(expectedHTML) {
				t.Errorf("content mismatch (first 500 chars):\n  got:  %s\n  want: %s",
					truncate(gotHTML, 500), truncate(expectedHTML, 500))
			}
		})
	}
}

var (
	rxMultiSpace      = regexp.MustCompile(`\s+`)
	rxSpaceBetweenTag = regexp.MustCompile(`>\s+<`)
	rxSelfClosing     = regexp.MustCompile(`\s*/>`)
)

func normalizeHTML(s string) string {
	// Normalize HTML entity differences between Go and JS renderers
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#34;", `"`)

	// Normalize void element self-closing tags: <br /> -> <br>, <img ... /> -> <img ...>
	s = rxSelfClosing.ReplaceAllString(s, ">")

	// Collapse whitespace between tags
	s = rxSpaceBetweenTag.ReplaceAllString(s, "><")

	// Collapse all remaining whitespace sequences to a single space
	s = rxMultiSpace.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
