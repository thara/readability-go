package readability

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func tagName(n *html.Node) string {
	if n == nil || n.Type != html.ElementNode {
		return ""
	}
	return strings.ToUpper(n.Data)
}

func getAttr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	if n == nil {
		return
	}
	for i, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func removeAttr(n *html.Node, key string) {
	if n == nil {
		return
	}
	attrs := n.Attr[:0]
	for _, a := range n.Attr {
		if !strings.EqualFold(a.Key, key) {
			attrs = append(attrs, a)
		}
	}
	n.Attr = attrs
}

func hasAttr(n *html.Node, key string) bool {
	if n == nil {
		return false
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return true
		}
	}
	return false
}

func className(n *html.Node) string {
	return getAttr(n, "class")
}

func nodeID(n *html.Node) string {
	return getAttr(n, "id")
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

func innerHTML(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		html.Render(&sb, c)
	}
	return sb.String()
}

func outerHTML(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	html.Render(&sb, n)
	return sb.String()
}

func children(n *html.Node) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			result = append(result, c)
		}
	}
	return result
}

func childNodes(n *html.Node) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, c)
	}
	return result
}

func firstElementChild(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return c
		}
	}
	return nil
}

func nextElementSibling(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

func previousElementSibling(n *html.Node) *html.Node {
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

func getElementsByTagName(n *html.Node, tag string) []*html.Node {
	tag = strings.ToLower(tag)
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (tag == "*" || c.Data == tag) {
			result = append(result, c)
		}
		result = append(result, getElementsByTagName(c, tag)...)
	}
	return result
}

func getAllNodesWithTag(n *html.Node, tags []string) []*html.Node {
	var result []*html.Node
	for _, tag := range tags {
		result = append(result, getElementsByTagName(n, tag)...)
	}
	return result
}

func removeNode(n *html.Node) {
	if n != nil && n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}

func replaceNode(oldNode, newNode *html.Node) {
	if oldNode.Parent != nil {
		if newNode.Parent != nil {
			newNode.Parent.RemoveChild(newNode)
		}
		oldNode.Parent.InsertBefore(newNode, oldNode)
		oldNode.Parent.RemoveChild(oldNode)
	}
}

func appendChild(parent, child *html.Node) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	parent.AppendChild(child)
}

func insertBefore(parent, newChild, reference *html.Node) {
	if newChild.Parent != nil {
		newChild.Parent.RemoveChild(newChild)
	}
	parent.InsertBefore(newChild, reference)
}

func createElement(tag string) *html.Node {
	return &html.Node{
		Type:     html.ElementNode,
		Data:     tag,
		DataAtom: atom.Lookup([]byte(tag)),
	}
}

func createTextNode(text string) *html.Node {
	return &html.Node{
		Type: html.TextNode,
		Data: text,
	}
}

func setNodeTag(n *html.Node, tag string) *html.Node {
	n.Data = strings.ToLower(tag)
	n.DataAtom = atom.Lookup([]byte(n.Data))
	return n
}

func removeNodes(nodes []*html.Node, filterFn func(*html.Node) bool) {
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if node.Parent != nil {
			if filterFn == nil || filterFn(node) {
				node.Parent.RemoveChild(node)
			}
		}
	}
}

func replaceNodeTags(nodes []*html.Node, newTag string) {
	for _, node := range nodes {
		setNodeTag(node, newTag)
	}
}

func getNextNode(node *html.Node, ignoreSelfAndKids bool) *html.Node {
	if !ignoreSelfAndKids {
		if fc := firstElementChild(node); fc != nil {
			return fc
		}
	}
	if ns := nextElementSibling(node); ns != nil {
		return ns
	}
	for node = node.Parent; node != nil; node = node.Parent {
		if ns := nextElementSibling(node); ns != nil {
			return ns
		}
	}
	return nil
}

func removeAndGetNext(node *html.Node) *html.Node {
	next := getNextNode(node, true)
	removeNode(node)
	return next
}

func getNodeAncestors(node *html.Node, maxDepth int) []*html.Node {
	var ancestors []*html.Node
	i := 0
	for node = node.Parent; node != nil; node = node.Parent {
		ancestors = append(ancestors, node)
		i++
		if maxDepth > 0 && i == maxDepth {
			break
		}
	}
	return ancestors
}

func isElementNode(n *html.Node) bool {
	return n != nil && n.Type == html.ElementNode
}

func isTextNode(n *html.Node) bool {
	return n != nil && n.Type == html.TextNode
}

func isPhrasingContent(n *html.Node) bool {
	if n.Type == html.TextNode {
		return true
	}
	tag := tagName(n)
	if phrasingElems[strings.ToLower(tag)] {
		return true
	}
	if tag == "A" || tag == "DEL" || tag == "INS" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !isPhrasingContent(c) {
				return false
			}
		}
		return true
	}
	return false
}

func isWhitespace(n *html.Node) bool {
	if n.Type == html.TextNode && strings.TrimSpace(n.Data) == "" {
		return true
	}
	if n.Type == html.ElementNode && n.Data == "br" {
		return true
	}
	return false
}

func isElementWithoutContent(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	text := strings.TrimSpace(textContent(n))
	if len(text) != 0 {
		return false
	}
	ch := children(n)
	if len(ch) == 0 {
		return true
	}
	brCount := len(getElementsByTagName(n, "br"))
	hrCount := len(getElementsByTagName(n, "hr"))
	return len(ch) == brCount+hrCount
}

func hasSingleTagInsideElement(element *html.Node, tag string) bool {
	ch := children(element)
	if len(ch) != 1 || tagName(ch[0]) != strings.ToUpper(tag) {
		return false
	}
	for c := element.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && rxHasContent.MatchString(c.Data) {
			return false
		}
	}
	return true
}

func hasChildBlockElement(element *html.Node) bool {
	for c := element.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if divToP[c.Data] || hasChildBlockElement(c) {
				return true
			}
		}
	}
	return false
}

func hasAncestorTag(node *html.Node, tag string, maxDepth int, filterFn func(*html.Node) bool) bool {
	if maxDepth == 0 {
		maxDepth = 3
	}
	tag = strings.ToLower(tag)
	depth := 0
	for node = node.Parent; node != nil; node = node.Parent {
		if maxDepth > 0 && depth > maxDepth {
			return false
		}
		if node.Type == html.ElementNode && node.Data == tag {
			if filterFn == nil || filterFn(node) {
				return true
			}
		}
		depth++
	}
	return false
}

func getInnerText(n *html.Node, normalizeSpaces bool) string {
	text := strings.TrimSpace(textContent(n))
	if normalizeSpaces {
		text = rxNormalize.ReplaceAllString(text, " ")
	}
	return text
}

func nextNonWhitespaceNode(n *html.Node) *html.Node {
	next := n
	for next != nil && next.Type != html.ElementNode && rxWhitespace.MatchString(textContent(next)) {
		next = next.NextSibling
	}
	return next
}

func isSingleImage(node *html.Node) bool {
	for node != nil {
		if tagName(node) == "IMG" {
			return true
		}
		ch := children(node)
		if len(ch) != 1 || strings.TrimSpace(textContent(node)) != "" {
			// Check if the direct text nodes (excluding children elements) are empty
			hasText := false
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode && strings.TrimSpace(c.Data) != "" {
					hasText = true
					break
				}
			}
			if hasText || len(ch) != 1 {
				return false
			}
		}
		node = ch[0]
	}
	return false
}

func documentBody(doc *html.Node) *html.Node {
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "body" {
			return c
		}
		if found := documentBody(c); found != nil {
			return found
		}
	}
	return nil
}

func removeCommentNodes(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.CommentNode {
			n.RemoveChild(c)
		} else {
			removeCommentNodes(c)
		}
		c = next
	}
}

func documentElement(doc *html.Node) *html.Node {
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "html" {
			return c
		}
	}
	return nil
}

func documentTitle(doc *html.Node) string {
	titles := getElementsByTagName(doc, "title")
	if len(titles) > 0 {
		return getInnerText(titles[0], true)
	}
	return ""
}
