package readability

import (
	"bytes"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

const (
	defaultMaxElemsToParse = 0
	defaultNTopCandidates  = 5
	defaultCharThreshold   = 500
)

type parseFlag int

const (
	flagStripUnlikelys     parseFlag = 1 << iota // 0x1
	flagWeightClasses                            // 0x2
	flagCleanConditionally                       // 0x4
)

type parseAttempt struct {
	articleContent *html.Node
	textLength     int
}

type parser struct {
	doc     *html.Node
	rawHTML []byte

	articleTitle    string
	articleByline   string
	articleDir      string
	articleLang     string
	articleSiteName string
	metadata        articleMetadata
	flags           parseFlag
	attempts        []parseAttempt
	scores          map[*html.Node]*contentScore
	dataTableFlags  map[*html.Node]bool

	// options
	maxElemsToParse     int
	nbTopCandidates     int
	charThreshold       int
	classesToPreserve   []string
	keepClasses         bool
	disableJSONLD       bool
	allowedVideoRegex   *regexp.Regexp
	linkDensityModifier float64
	baseURI             string
	documentURI         string
	debugWriter         io.Writer
}

func newParser(documentURI string, opts ...Option) *parser {
	p := &parser{
		flags:             flagStripUnlikelys | flagWeightClasses | flagCleanConditionally,
		scores:            make(map[*html.Node]*contentScore),
		dataTableFlags:    make(map[*html.Node]bool),
		maxElemsToParse:   defaultMaxElemsToParse,
		nbTopCandidates:   defaultNTopCandidates,
		charThreshold:     defaultCharThreshold,
		classesToPreserve: []string{"page"},
		allowedVideoRegex: rxVideos,
		documentURI:       documentURI,
		baseURI:           documentURI,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *parser) flagIsActive(flag parseFlag) bool {
	return p.flags&flag != 0
}

func (p *parser) removeFlag(flag parseFlag) {
	p.flags = p.flags &^ flag
}

func (p *parser) reparseDoc() {
	doc, err := html.Parse(bytes.NewReader(p.rawHTML))
	if err != nil {
		return
	}
	p.doc = doc
	p.scores = make(map[*html.Node]*contentScore)
	p.dataTableFlags = make(map[*html.Node]bool)
}

func (p *parser) run() (*Article, error) {
	if p.maxElemsToParse > 0 {
		allElems := getElementsByTagName(p.doc, "*")
		if len(allElems) > p.maxElemsToParse {
			return nil, ErrTooManyElements
		}
	}

	p.unwrapNoscriptImages()

	var jsonld articleMetadata
	if !p.disableJSONLD {
		jsonld = p.getJSONLD()
	}

	p.removeScripts()
	p.prepDocument()

	metadata := p.getArticleMetadata(jsonld)
	p.metadata = metadata
	p.articleTitle = metadata.title

	baseFromDoc := findDocumentBaseURI(p.doc)
	if baseFromDoc != "" {
		docBase, err := url.Parse(p.documentURI)
		if err == nil {
			ref, err := url.Parse(baseFromDoc)
			if err == nil {
				p.baseURI = docBase.ResolveReference(ref).String()
			} else {
				p.baseURI = baseFromDoc
			}
		} else {
			p.baseURI = baseFromDoc
		}
	}

	articleContent := p.grabArticle()
	if articleContent == nil {
		return nil, nil
	}

	p.postProcessContent(articleContent)

	if metadata.excerpt == "" {
		paragraphs := getElementsByTagName(articleContent, "p")
		if len(paragraphs) > 0 {
			metadata.excerpt = strings.TrimSpace(textContent(paragraphs[0]))
		}
	}

	tc := textContent(articleContent)
	return &Article{
		Title:         p.articleTitle,
		Byline:        firstNonEmpty(metadata.byline, p.articleByline),
		Dir:           p.articleDir,
		Lang:          p.articleLang,
		Content:       innerHTML(articleContent),
		TextContent:   tc,
		Length:        len(tc),
		Excerpt:       metadata.excerpt,
		SiteName:      firstNonEmpty(metadata.siteName, p.articleSiteName),
		PublishedTime: metadata.publishedTime,
	}, nil
}
