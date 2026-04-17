# readability-go

A Go port of [Mozilla's Readability.js](https://github.com/mozilla/readability) (v0.6.0) — the library that powers Firefox Reader View. It extracts the main readable content from an HTML page.

## Installation

```
go get github.com/thara/readability-go
```

## Usage

### Extracting article content

```go
package main

import (
	"fmt"
	"strings"

	readability "github.com/thara/readability-go"
)

func main() {
	html := `<html><head><title>Example</title></head>
	<body><article><h1>Hello</h1><p>This is the article content.</p></article></body></html>`

	article, err := readability.Parse(
		strings.NewReader(html),
		"https://example.com/article",
	)
	if err != nil {
		panic(err)
	}
	if article == nil {
		fmt.Println("No article content found")
		return
	}

	fmt.Println("Title:", article.Title)
	fmt.Println("Content:", article.Content)
}
```

The returned `Article` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| Title | string | Article title |
| Byline | string | Author information |
| Content | string | Article HTML content |
| TextContent | string | Article text content (no HTML) |
| Length | int | Length of TextContent |
| Excerpt | string | Article excerpt / description |
| Dir | string | Text direction ("ltr" or "rtl") |
| Lang | string | Content language |
| SiteName | string | Site name |
| PublishedTime | string | Published time |

### Checking if a page is readable

```go
readable, err := readability.IsProbablyReaderable(strings.NewReader(html))
if err != nil {
	panic(err)
}
fmt.Println("Readable:", readable)
```

## Options

### Parse options

```go
article, err := readability.Parse(r, url,
	readability.WithCharThreshold(500),
	readability.WithClassesToPreserve([]string{"caption", "credit"}),
	readability.WithKeepClasses(true),
)
```

| Option | Description |
|--------|-------------|
| `WithMaxElemsToParse(n int)` | Maximum number of elements to parse (0 = no limit) |
| `WithNbTopCandidates(n int)` | Number of top candidates to consider (default: 5) |
| `WithCharThreshold(n int)` | Minimum character count for an article (default: 500) |
| `WithClassesToPreserve([]string)` | CSS classes to keep during cleanup |
| `WithKeepClasses(bool)` | Keep all CSS classes |
| `WithDisableJSONLD(bool)` | Disable JSON-LD metadata extraction |
| `WithAllowedVideoRegex(*regexp.Regexp)` | Regex for allowed video URLs |
| `WithLinkDensityModifier(float64)` | Adjust link density threshold |
| `WithDebug(io.Writer)` | Enable debug logging |

### IsProbablyReaderable options

| Option | Description |
|--------|-------------|
| `WithMinScore(float64)` | Minimum score to be considered readable (default: 20) |
| `WithMinContentLength(int)` | Minimum node content length (default: 140) |
| `WithVisibilityChecker(func(*html.Node) bool)` | Custom visibility checker |

## CLI Tool

### Install

```
go install github.com/thara/readability-go/cmd/readability@latest
```

### Usage

```
readability [options] [URL | FILE | -]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-base-url` | Base URL for resolving relative links |
| `-json` | Output as JSON |
| `-check` | Only check if the document is probably readable |

**Examples:**

```bash
# From URL
readability https://example.com/article

# From file
readability article.html

# From stdin
curl -s https://example.com/article | readability -

# JSON output
readability -json https://example.com/article

# Check readability
readability -check https://example.com/article
```

## Compatibility

Based on Readability.js v0.6.0. Passes 127 of 130 Mozilla test fixtures. The 3 skipped tests are caused by structural differences between Go's `net/html` parser and Mozilla's JSDOMParser:

- **hukumusume** — Table-based layouts produce fundamentally different DOM trees
- **nytimes-5** — Slight scoring divergence due to DOM structure differences
- **wikipedia-2** — Minor 11-character difference in 370K of output (Go collapses `<div><p>` to `<p>` inside `<th>`)
