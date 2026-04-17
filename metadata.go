package readability

import (
	"encoding/json"
	"html"
	"strings"

	nethtml "golang.org/x/net/html"
)

type articleMetadata struct {
	title         string
	byline        string
	excerpt       string
	siteName      string
	publishedTime string
}

func unescapeHTMLEntities(s string) string {
	if s == "" {
		return s
	}
	return html.UnescapeString(s)
}

func (p *parser) getJSONLD() articleMetadata {
	var metadata articleMetadata
	scripts := getAllNodesWithTag(p.doc, []string{"script"})
	found := false

	for _, script := range scripts {
		if found {
			break
		}
		if getAttr(script, "type") != "application/ld+json" {
			continue
		}

		content := rxCDATA.ReplaceAllString(textContent(script), "")

		var parsed interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			continue
		}

		parsed = p.findJSONLDArticle(parsed)
		if parsed == nil {
			continue
		}

		obj, ok := parsed.(map[string]interface{})
		if !ok {
			continue
		}

		ctx := obj["@context"]
		switch v := ctx.(type) {
		case string:
			if !rxSchemaOrg.MatchString(v) {
				continue
			}
		case map[string]interface{}:
			vocab, _ := v["@vocab"].(string)
			if !rxSchemaOrg.MatchString(vocab) {
				continue
			}
		default:
			continue
		}

		atType, _ := obj["@type"].(string)
		if atType == "" {
			graph, ok := obj["@graph"].([]interface{})
			if !ok {
				continue
			}
			obj = nil
			for _, item := range graph {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				t, _ := m["@type"].(string)
				if rxJSONLdArticleTypes.MatchString(t) {
					obj = m
					break
				}
			}
			if obj == nil {
				continue
			}
			atType, _ = obj["@type"].(string)
		}

		if !rxJSONLdArticleTypes.MatchString(atType) {
			continue
		}

		found = true

		name, _ := obj["name"].(string)
		headline, _ := obj["headline"].(string)

		if name != "" && headline != "" && name != headline {
			title := p.getArticleTitle()
			nameMatches := textSimilarity(name, title) > 0.75
			headlineMatches := textSimilarity(headline, title) > 0.75
			if headlineMatches && !nameMatches {
				metadata.title = headline
			} else {
				metadata.title = name
			}
		} else if name != "" {
			metadata.title = strings.TrimSpace(name)
		} else if headline != "" {
			metadata.title = strings.TrimSpace(headline)
		}

		if author := obj["author"]; author != nil {
			switch a := author.(type) {
			case map[string]interface{}:
				if authorName, ok := a["name"].(string); ok {
					metadata.byline = strings.TrimSpace(authorName)
				}
			case []interface{}:
				var names []string
				for _, item := range a {
					if m, ok := item.(map[string]interface{}); ok {
						if authorName, ok := m["name"].(string); ok {
							names = append(names, strings.TrimSpace(authorName))
						}
					}
				}
				if len(names) > 0 {
					metadata.byline = strings.Join(names, ", ")
				}
			}
		}

		if desc, ok := obj["description"].(string); ok {
			metadata.excerpt = strings.TrimSpace(desc)
		}
		if pub, ok := obj["publisher"].(map[string]interface{}); ok {
			if pubName, ok := pub["name"].(string); ok {
				metadata.siteName = strings.TrimSpace(pubName)
			}
		}
		if datePublished, ok := obj["datePublished"].(string); ok {
			metadata.publishedTime = strings.TrimSpace(datePublished)
		}
	}

	return metadata
}

func (p *parser) findJSONLDArticle(parsed interface{}) interface{} {
	switch v := parsed.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				t, _ := m["@type"].(string)
				if rxJSONLdArticleTypes.MatchString(t) {
					return m
				}
			}
		}
		return nil
	default:
		return v
	}
}

func (p *parser) getArticleMetadata(jsonld articleMetadata) articleMetadata {
	var metadata articleMetadata
	values := make(map[string]string)

	metaElements := getElementsByTagName(p.doc, "meta")

	for _, element := range metaElements {
		elementName := getAttr(element, "name")
		elementProperty := getAttr(element, "property")
		content := getAttr(element, "content")
		if content == "" {
			continue
		}

		if elementProperty != "" {
			matches := rxPropertyPattern.FindStringSubmatch(elementProperty)
			if matches != nil {
				name := strings.ToLower(strings.ReplaceAll(elementProperty, " ", ""))
				name = strings.TrimSpace(name)
				// Re-match to get the cleaned version
				cleaned := rxPropertyPattern.FindString(elementProperty)
				if cleaned != "" {
					name = strings.ToLower(strings.ReplaceAll(cleaned, " ", ""))
				}
				values[name] = strings.TrimSpace(content)
			}
		}

		if elementName != "" && rxNamePattern.MatchString(elementName) {
			name := strings.ToLower(strings.ReplaceAll(elementName, " ", ""))
			name = strings.ReplaceAll(name, ".", ":")
			values[name] = strings.TrimSpace(content)
		}
	}

	metadata.title = firstNonEmpty(
		jsonld.title,
		values["dc:title"],
		values["dcterm:title"],
		values["og:title"],
		values["weibo:article:title"],
		values["weibo:webpage:title"],
		values["title"],
		values["twitter:title"],
		values["parsely-title"],
	)
	if metadata.title == "" {
		metadata.title = p.getArticleTitle()
	}

	articleAuthor := values["article:author"]
	if articleAuthor != "" && isURL(articleAuthor) {
		articleAuthor = ""
	}

	metadata.byline = firstNonEmpty(
		jsonld.byline,
		values["dc:creator"],
		values["dcterm:creator"],
		values["author"],
		values["parsely-author"],
		articleAuthor,
	)

	metadata.excerpt = firstNonEmpty(
		jsonld.excerpt,
		values["dc:description"],
		values["dcterm:description"],
		values["og:description"],
		values["weibo:article:description"],
		values["weibo:webpage:description"],
		values["description"],
		values["twitter:description"],
	)

	metadata.siteName = firstNonEmpty(
		jsonld.siteName,
		values["og:site_name"],
	)

	metadata.publishedTime = firstNonEmpty(
		jsonld.publishedTime,
		values["article:published_time"],
		values["parsely-pub-date"],
	)

	metadata.title = unescapeHTMLEntities(metadata.title)
	metadata.byline = unescapeHTMLEntities(metadata.byline)
	metadata.excerpt = unescapeHTMLEntities(metadata.excerpt)
	metadata.siteName = unescapeHTMLEntities(metadata.siteName)
	metadata.publishedTime = unescapeHTMLEntities(metadata.publishedTime)

	return metadata
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func findDocumentBaseURI(doc *nethtml.Node) string {
	bases := getElementsByTagName(doc, "base")
	for _, base := range bases {
		href := getAttr(base, "href")
		if href != "" {
			return href
		}
	}
	return ""
}
