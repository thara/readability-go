package readability

import (
	"io"
	"math"
	"strings"

	"golang.org/x/net/html"
)

type readerableConfig struct {
	minScore          float64
	minContentLength  int
	visibilityChecker func(*html.Node) bool
}

// ReaderableOption configures the IsProbablyReaderable function.
type ReaderableOption func(*readerableConfig)

// WithMinScore sets the minimum cumulated score for the document to be considered readable.
func WithMinScore(score float64) ReaderableOption {
	return func(c *readerableConfig) {
		c.minScore = score
	}
}

// WithMinContentLength sets the minimum node content length.
func WithMinContentLength(length int) ReaderableOption {
	return func(c *readerableConfig) {
		c.minContentLength = length
	}
}

// WithVisibilityChecker sets the function used to determine if a node is visible.
func WithVisibilityChecker(fn func(*html.Node) bool) ReaderableOption {
	return func(c *readerableConfig) {
		c.visibilityChecker = fn
	}
}

// IsNodeVisible is the default visibility checker for IsProbablyReaderable.
func IsNodeVisible(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return true
	}

	style := getAttr(node, "style")
	if style != "" && strings.Contains(strings.ToLower(style), "display:none") || strings.Contains(strings.ReplaceAll(strings.ToLower(style), " ", ""), "display:none") {
		return false
	}

	if hasAttr(node, "hidden") {
		return false
	}

	ariaHidden := getAttr(node, "aria-hidden")
	if ariaHidden == "true" {
		cls := className(node)
		if !strings.Contains(cls, "fallback-image") {
			return false
		}
	}

	return true
}

// IsProbablyReaderable checks whether a document is likely to be readable
// without performing the full parse.
func IsProbablyReaderable(r io.Reader, opts ...ReaderableOption) (bool, error) {
	cfg := &readerableConfig{
		minScore:          20,
		minContentLength:  140,
		visibilityChecker: IsNodeVisible,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	doc, err := html.Parse(r)
	if err != nil {
		return false, err
	}

	nodes := getAllNodesWithTag(doc, []string{"p", "pre", "article"})

	brNodes := getAllNodesWithTag(doc, []string{"br"})
	seen := make(map[*html.Node]bool)
	for _, n := range nodes {
		seen[n] = true
	}
	for _, br := range brNodes {
		if br.Parent != nil && br.Parent.Type == html.ElementNode && br.Parent.Data == "div" {
			if !seen[br.Parent] {
				seen[br.Parent] = true
				nodes = append(nodes, br.Parent)
			}
		}
	}

	score := float64(0)

	for _, node := range nodes {
		if !cfg.visibilityChecker(node) {
			continue
		}

		matchString := className(node) + " " + nodeID(node)
		if rxUnlikelyCandidates.MatchString(matchString) && !rxOkMaybeItsACandidate.MatchString(matchString) {
			continue
		}

		if hasLiAncestor(node) {
			continue
		}

		tc := strings.TrimSpace(textContent(node))
		tcLen := len(tc)
		if tcLen < cfg.minContentLength {
			continue
		}

		score += math.Sqrt(float64(tcLen - cfg.minContentLength))
		if score > cfg.minScore {
			return true, nil
		}
	}

	return false, nil
}

func hasLiAncestor(node *html.Node) bool {
	if node.Type != html.ElementNode || node.Data != "p" {
		return false
	}
	for p := node.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "li" {
			return true
		}
	}
	return false
}
