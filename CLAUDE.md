# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go port of Mozilla's Readability.js (v0.6.0) — the library behind Firefox Reader View. Extracts readable article content from HTML documents. Single external dependency: `golang.org/x/net/html`.

## Build & Test

```bash
go build ./...              # Build all packages including CLI
go test ./...               # Run all tests (130 Mozilla fixture tests)
go test -run TestReadabilityFixtures/001 -v  # Run a single fixture test
go vet ./...                # Static analysis
```

The CLI tool lives in `cmd/readability/`:
```bash
go run ./cmd/readability/ https://example.com        # Fetch and extract
echo '<html>...' | go run ./cmd/readability/          # From stdin
go run ./cmd/readability/ -json testdata/test-pages/001/source.html  # JSON output
go run ./cmd/readability/ -check https://example.com  # Readability check only
```

## Architecture

The `Parse()` pipeline flows through these stages:

1. **Input** (`readability.go`) — Read HTML, parse DOM via `html.Parse`, strip comment nodes
2. **Pre-process** (`cleaning.go`) — Unwrap noscript images, remove scripts/styles, convert multi-BR to P, replace `<font>` with `<span>`
3. **Metadata** (`metadata.go`, `title.go`) — Extract from JSON-LD (`<script type="application/ld+json">`) and `<meta>` tags (OG, Twitter, DC)
4. **Content extraction** (`scoring.go: grabArticle()`) — The core algorithm: walk DOM removing unlikely candidates, score nodes by text/comma density with ancestor propagation, select top candidate, collect qualifying siblings, then clean via `prepArticle()`
5. **Retry loop** (`parser.go`) — If extracted text < `charThreshold` (500), progressively disable flags (`stripUnlikelys` → `weightClasses` → `cleanConditionally`) and re-parse from raw HTML
6. **Post-process** (`postprocess.go`) — Resolve relative URIs, simplify nested div/section wrappers, strip non-preserved CSS classes

### Key internal design decisions

- **Scoring storage**: `map[*html.Node]*contentScore` replaces JS's `node.readability.contentScore`
- **Data table flags**: `map[*html.Node]bool` replaces JS's `node._readabilityDataTable`
- **Tag mutation**: `setNodeTag()` modifies `n.Data` and `n.DataAtom` directly (no node recreation)
- **Re-parse on retry**: Raw HTML bytes are cached; on threshold failure, `html.Parse()` rebuilds the DOM since previous attempts mutate it destructively

### Support modules

- `node.go` — ~40 DOM helper functions wrapping `*html.Node` operations (traversal, mutation, query, predicates)
- `regexp.go` — 30 pre-compiled regex patterns for content classification
- `constants.go` — Tag/role lookup tables (`divToP`, `phrasingElems`, `unlikelyRoles`, etc.)

## Test Fixtures

Tests compare against Mozilla's 130 test cases in `testdata/test-pages/`. Each directory contains:
- `source.html` — Input HTML
- `expected.html` — Expected clean HTML output
- `expected-metadata.json` — Expected metadata (title, byline, excerpt, etc.)

All tests use `documentURI = "http://fakehost/test/page.html"` and `WithClassesToPreserve([]string{"caption"})` to match Mozilla's test runner configuration.

HTML comparison uses `normalizeHTML()` which handles known Go/JS serialization differences: entity unescaping, self-closing tags, `<tbody>` insertion, SVG attribute casing, whitespace normalization.

## Known Parser Differences

3 fixtures are skipped due to fundamental DOM structure differences between Go's `golang.org/x/net/html` (HTML5 spec parser) and Mozilla's JSDOMParser:
- `hukumusume` — Table-based layout produces entirely different DOM trees
- `nytimes-5` — Scoring divergence from structural differences causes different sibling selection
- `wikipedia-2` — Minor 11-char difference in div-to-P wrapper handling
