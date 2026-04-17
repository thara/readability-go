package readability

import (
	"errors"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ErrTooManyElements is returned when the document contains more elements than maxElemsToParse.
var ErrTooManyElements = errors.New("readability: too many elements to parse")

// Article is the result of parsing a document for its readable content.
type Article struct {
	Title         string `json:"title"`
	Byline        string `json:"byline"`
	Dir           string `json:"dir"`
	Lang          string `json:"lang"`
	Content       string `json:"content"`
	TextContent   string `json:"textContent"`
	Length        int    `json:"length"`
	Excerpt       string `json:"excerpt"`
	SiteName      string `json:"siteName"`
	PublishedTime string `json:"publishedTime"`
}

// Option configures the parser behavior.
type Option func(*parser)

// WithMaxElemsToParse sets the maximum number of elements to parse. 0 means no limit.
func WithMaxElemsToParse(n int) Option {
	return func(p *parser) { p.maxElemsToParse = n }
}

// WithNbTopCandidates sets the number of top candidates to consider.
func WithNbTopCandidates(n int) Option {
	return func(p *parser) { p.nbTopCandidates = n }
}

// WithCharThreshold sets the minimum number of characters an article must have.
func WithCharThreshold(n int) Option {
	return func(p *parser) { p.charThreshold = n }
}

// WithClassesToPreserve sets additional CSS classes to preserve during cleanup.
func WithClassesToPreserve(classes []string) Option {
	return func(p *parser) {
		p.classesToPreserve = append(p.classesToPreserve, classes...)
	}
}

// WithKeepClasses keeps all CSS classes if set to true.
func WithKeepClasses(keep bool) Option {
	return func(p *parser) { p.keepClasses = keep }
}

// WithDisableJSONLD disables JSON-LD metadata extraction.
func WithDisableJSONLD(disable bool) Option {
	return func(p *parser) { p.disableJSONLD = disable }
}

// WithAllowedVideoRegex sets the regex for allowed video URLs.
func WithAllowedVideoRegex(re *regexp.Regexp) Option {
	return func(p *parser) { p.allowedVideoRegex = re }
}

// WithLinkDensityModifier adjusts the link density threshold.
func WithLinkDensityModifier(mod float64) Option {
	return func(p *parser) { p.linkDensityModifier = mod }
}

// WithDebug enables debug logging to the given writer.
func WithDebug(w io.Writer) Option {
	return func(p *parser) { p.debugWriter = w }
}

// Parse extracts the main readable content from an HTML document.
// The input is read from r. The documentURI is used for resolving relative URLs.
// Returns nil Article if no content could be extracted.
func Parse(r io.Reader, documentURI string, opts ...Option) (*Article, error) {
	rawHTML, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(rawHTML)))
	if err != nil {
		return nil, err
	}

	removeCommentNodes(doc)

	p := newParser(documentURI, opts...)
	p.rawHTML = rawHTML
	p.doc = doc

	return p.run()
}
