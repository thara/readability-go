package readability

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func (p *parser) postProcessContent(articleContent *html.Node) {
	p.fixRelativeUris(articleContent)
	p.simplifyNestedElements(articleContent)
	if !p.keepClasses {
		p.cleanClasses(articleContent)
	}
}

func (p *parser) fixRelativeUris(articleContent *html.Node) {
	baseURI := p.baseURI
	documentURI := p.documentURI

	toAbsoluteURI := func(uri string) string {
		uri = strings.TrimSpace(uri)
		if baseURI == documentURI && len(uri) > 0 && uri[0] == '#' {
			return uri
		}
		base, err := url.Parse(baseURI)
		if err != nil {
			return uri
		}
		ref, err := url.Parse(uri)
		if err != nil {
			ref, err = url.Parse("./" + uri)
			if err != nil {
				return uri
			}
		}
		resolved := base.ResolveReference(ref)
		if resolved.Path == "" && resolved.Host != "" {
			resolved.Path = "/"
		}
		result := resolved.String()
		if strings.HasSuffix(uri, "#") && !strings.HasSuffix(result, "#") {
			result += "#"
		}
		return result
	}

	links := getAllNodesWithTag(articleContent, []string{"a"})
	for _, link := range links {
		href := getAttr(link, "href")
		if href == "" {
			continue
		}
		if strings.HasPrefix(href, "javascript:") {
			cns := childNodes(link)
			if len(cns) == 1 && cns[0].Type == html.TextNode {
				text := createTextNode(textContent(link))
				replaceNode(link, text)
			} else {
				container := createElement("span")
				for link.FirstChild != nil {
					appendChild(container, link.FirstChild)
				}
				replaceNode(link, container)
			}
		} else {
			setAttr(link, "href", toAbsoluteURI(href))
		}
	}

	medias := getAllNodesWithTag(articleContent, []string{"img", "picture", "figure", "video", "audio", "source"})
	for _, media := range medias {
		src := getAttr(media, "src")
		poster := getAttr(media, "poster")
		srcset := getAttr(media, "srcset")

		if src != "" {
			setAttr(media, "src", toAbsoluteURI(src))
		}
		if poster != "" {
			setAttr(media, "poster", toAbsoluteURI(poster))
		}
		if srcset != "" {
			newSrcset := rxSrcsetURL.ReplaceAllStringFunc(srcset, func(match string) string {
				parts := rxSrcsetURL.FindStringSubmatch(match)
				if len(parts) < 4 {
					return match
				}
				return toAbsoluteURI(parts[1]) + parts[2] + parts[3]
			})
			setAttr(media, "srcset", newSrcset)
		}
	}
}

func (p *parser) simplifyNestedElements(articleContent *html.Node) {
	node := articleContent
	for node != nil {
		tag := tagName(node)
		if node.Parent != nil && (tag == "DIV" || tag == "SECTION") {
			id := nodeID(node)
			if !(id != "" && strings.HasPrefix(id, "readability")) {
				if isElementWithoutContent(node) {
					node = removeAndGetNext(node)
					continue
				} else if hasSingleTagInsideElement(node, "DIV") || hasSingleTagInsideElement(node, "SECTION") {
					child := children(node)[0]
					for _, attr := range node.Attr {
						setAttr(child, attr.Key, attr.Val)
					}
					replaceNode(node, child)
					node = child
					continue
				}
			}
		}
		node = getNextNode(node, false)
	}
}

func (p *parser) cleanClasses(node *html.Node) {
	if node.Type != html.ElementNode {
		return
	}

	cls := getAttr(node, "class")
	if cls != "" {
		parts := strings.Fields(cls)
		var preserved []string
		for _, part := range parts {
			for _, keep := range p.classesToPreserve {
				if part == keep {
					preserved = append(preserved, part)
					break
				}
			}
		}
		if len(preserved) > 0 {
			setAttr(node, "class", strings.Join(preserved, " "))
		} else {
			removeAttr(node, "class")
		}
	}

	for c := firstElementChild(node); c != nil; c = nextElementSibling(c) {
		p.cleanClasses(c)
	}
}
