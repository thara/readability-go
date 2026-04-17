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

			article, err := Parse(strings.NewReader(string(sourceBytes)), "http://fakehost/test/page.html", WithClassesToPreserve([]string{"caption"}))
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

			gotExcerpt := strings.ReplaceAll(article.Excerpt, "\u00a0", " ")
			wantExcerpt := strings.ReplaceAll(strVal(meta.Excerpt), "&nbsp;", " ")
			wantExcerpt = strings.ReplaceAll(wantExcerpt, "\u00a0", " ")
			if gotExcerpt != wantExcerpt {
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

			normGot := normalizeHTML(gotHTML)
			normWant := normalizeHTML(expectedHTML)
			if normGot != normWant {
				// Find first divergence position
				minLen := len(normGot)
				if len(normWant) < minLen {
					minLen = len(normWant)
				}
				diffPos := minLen
				for i := 0; i < minLen; i++ {
					if normGot[i] != normWant[i] {
						diffPos = i
						break
					}
				}
				start := diffPos - 60
				if start < 0 {
					start = 0
				}
				gotSnip := normGot[start:]
				if len(gotSnip) > 120 {
					gotSnip = gotSnip[:120]
				}
				wantSnip := normWant[start:]
				if len(wantSnip) > 120 {
					wantSnip = wantSnip[:120]
				}
				t.Errorf("content mismatch at pos %d (len got=%d want=%d):\n  got:  ...%s...\n  want: ...%s...",
					diffPos, len(normGot), len(normWant), gotSnip, wantSnip)
			}
		})
	}
}

var (
	rxMultiSpace      = regexp.MustCompile(`\s+`)
	rxSpaceBetweenTag = regexp.MustCompile(`>\s+<`)
	rxSelfClosing     = regexp.MustCompile(`\s*/>`)
	rxHTMLComment     = regexp.MustCompile(`<!--[\s\S]*?-->`)
	rxEmptyClass      = regexp.MustCompile(` class=""`)
	rxTbody           = regexp.MustCompile(`</?tbody>`)
	rxSpaceBeforeClose = regexp.MustCompile(`\s+(</(?:p|li|td|th|div|h[1-6])>)`)
)

var svgCaseReplacer = strings.NewReplacer(
	"viewBox", "viewbox",
	"clipPath", "clippath",
	"clipPathUnits", "clippathunits",
	"fillOpacity", "fillopacity",
	"gradientTransform", "gradienttransform",
	"gradientUnits", "gradientunits",
	"markerHeight", "markerheight",
	"markerWidth", "markerwidth",
	"patternContentUnits", "patterncontentunits",
	"patternTransform", "patterntransform",
	"patternUnits", "patternunits",
	"preserveAspectRatio", "preserveaspectratio",
	"spreadMethod", "spreadmethod",
	"stopColor", "stopcolor",
	"stopOpacity", "stopopacity",
	"strokeDasharray", "strokedasharray",
	"strokeDashoffset", "strokedashoffset",
	"strokeLinecap", "strokelinecap",
	"strokeLinejoin", "strokelinejoin",
	"strokeMiterlimit", "strokemiterlimit",
	"strokeOpacity", "strokeopacity",
	"strokeWidth", "strokewidth",
	"textAnchor", "textanchor",
	"textDecoration", "textdecoration",
	"textRendering", "textrendering",
)

func normalizeHTML(s string) string {
	s = rxHTMLComment.ReplaceAllString(s, "")

	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#34;", `"`)
	s = strings.ReplaceAll(s, "&apos;", "'")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&#160;", " ")
	s = strings.ReplaceAll(s, "\u00a0", " ")

	s = rxSelfClosing.ReplaceAllString(s, ">")

	s = rxEmptyClass.ReplaceAllString(s, "")

	s = rxTbody.ReplaceAllString(s, "")

	s = svgCaseReplacer.Replace(s)

	s = rxSpaceBeforeClose.ReplaceAllString(s, "$1")

	s = rxSpaceBetweenTag.ReplaceAllString(s, "><")

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
