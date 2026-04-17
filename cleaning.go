package readability

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func (p *parser) removeScripts() {
	removeNodes(getAllNodesWithTag(p.doc, []string{"script", "noscript"}), nil)
}

func (p *parser) prepDocument() {
	removeNodes(getAllNodesWithTag(p.doc, []string{"style"}), nil)

	body := documentBody(p.doc)
	if body != nil {
		p.replaceBrs(body)
	}

	replaceNodeTags(getAllNodesWithTag(p.doc, []string{"font"}), "span")
}

func (p *parser) replaceBrs(elem *html.Node) {
	brs := getAllNodesWithTag(elem, []string{"br"})
	for _, br := range brs {
		next := br.NextSibling

		replaced := false

		for {
			next = nextNonWhitespaceNode(next)
			if next == nil || tagName(next) != "BR" {
				break
			}
			replaced = true
			brSibling := next.NextSibling
			removeNode(next)
			next = brSibling
		}

		if replaced {
			newP := createElement("p")
			replaceNode(br, newP)

			next = newP.NextSibling
			for next != nil {
				if tagName(next) == "BR" {
					nextElem := nextNonWhitespaceNode(next.NextSibling)
					if nextElem != nil && tagName(nextElem) == "BR" {
						break
					}
				}

				if !isPhrasingContent(next) {
					break
				}

				sibling := next.NextSibling
				appendChild(newP, next)
				next = sibling
			}

			for newP.LastChild != nil && isWhitespace(newP.LastChild) {
				removeNode(newP.LastChild)
			}

			if tagName(newP.Parent) == "P" {
				setNodeTag(newP.Parent, "div")
			}
		}
	}
}

func (p *parser) unwrapNoscriptImages() {
	imgs := getElementsByTagName(p.doc, "img")
	for _, img := range imgs {
		hasImageAttr := false
		for _, attr := range img.Attr {
			switch attr.Key {
			case "src", "srcset", "data-src", "data-srcset":
				hasImageAttr = true
			default:
				if rxImageExtension.MatchString(attr.Val) {
					hasImageAttr = true
				}
			}
			if hasImageAttr {
				break
			}
		}
		if !hasImageAttr {
			removeNode(img)
		}
	}

	noscripts := getElementsByTagName(p.doc, "noscript")
	for _, noscript := range noscripts {
		if !isSingleImage(noscript) {
			continue
		}

		tmp := createElement("div")
		tmpContent := textContent(noscript)
		// Parse the noscript innerHTML
		fragment, err := html.ParseFragment(strings.NewReader(tmpContent), &html.Node{
			Type:     html.ElementNode,
			Data:     "div",
			DataAtom: 0,
		})
		if err != nil || len(fragment) == 0 {
			continue
		}
		for _, f := range fragment {
			appendChild(tmp, f)
		}

		prevElement := previousElementSibling(noscript)
		if prevElement != nil && isSingleImage(prevElement) {
			prevImg := prevElement
			if tagName(prevImg) != "IMG" {
				imgs := getElementsByTagName(prevElement, "img")
				if len(imgs) > 0 {
					prevImg = imgs[0]
				}
			}

			newImgs := getElementsByTagName(tmp, "img")
			if len(newImgs) == 0 {
				continue
			}
			newImg := newImgs[0]

			for _, attr := range prevImg.Attr {
				if attr.Val == "" {
					continue
				}
				if attr.Key == "src" || attr.Key == "srcset" || rxImageExtension.MatchString(attr.Val) {
					if getAttr(newImg, attr.Key) == attr.Val {
						continue
					}
					attrName := attr.Key
					if hasAttr(newImg, attrName) {
						attrName = "data-old-" + attrName
					}
					setAttr(newImg, attrName, attr.Val)
				}
			}

			fc := firstElementChild(tmp)
			if fc != nil {
				replaceNode(prevElement, fc)
			}
		}
	}
}

func (p *parser) prepArticle(articleContent *html.Node) {
	p.cleanStyles(articleContent)
	p.markDataTables(articleContent)
	p.fixLazyImages(articleContent)

	p.cleanConditionally(articleContent, "form")
	p.cleanConditionally(articleContent, "fieldset")
	p.clean(articleContent, "object")
	p.clean(articleContent, "embed")
	p.clean(articleContent, "footer")
	p.clean(articleContent, "link")
	p.clean(articleContent, "aside")

	shareElementThreshold := defaultCharThreshold
	for _, topCandidate := range children(articleContent) {
		p.cleanMatchedNodes(topCandidate, func(node *html.Node, matchString string) bool {
			return rxShareElements.MatchString(matchString) &&
				len(textContent(node)) < shareElementThreshold
		})
	}

	p.clean(articleContent, "iframe")
	p.clean(articleContent, "input")
	p.clean(articleContent, "textarea")
	p.clean(articleContent, "select")
	p.clean(articleContent, "button")
	p.cleanHeaders(articleContent)

	p.cleanConditionally(articleContent, "table")
	p.cleanConditionally(articleContent, "ul")
	p.cleanConditionally(articleContent, "div")

	replaceNodeTags(getAllNodesWithTag(articleContent, []string{"h1"}), "h2")

	removeNodes(getAllNodesWithTag(articleContent, []string{"p"}), func(paragraph *html.Node) bool {
		imgCount := len(getAllNodesWithTag(paragraph, []string{"img", "embed", "object", "iframe"}))
		return imgCount == 0 && getInnerText(paragraph, false) == ""
	})

	brs := getAllNodesWithTag(articleContent, []string{"br"})
	for _, br := range brs {
		next := nextNonWhitespaceNode(br.NextSibling)
		if next != nil && tagName(next) == "P" {
			removeNode(br)
		}
	}

	tables := getAllNodesWithTag(articleContent, []string{"table"})
	for _, table := range tables {
		var tbody *html.Node
		if hasSingleTagInsideElement(table, "TBODY") {
			tbody = firstElementChild(table)
		} else {
			tbody = table
		}
		if hasSingleTagInsideElement(tbody, "TR") {
			row := firstElementChild(tbody)
			if hasSingleTagInsideElement(row, "TD") {
				cell := firstElementChild(row)
				newTag := "div"
				allPhrasing := true
				for c := cell.FirstChild; c != nil; c = c.NextSibling {
					if !isPhrasingContent(c) {
						allPhrasing = false
						break
					}
				}
				if allPhrasing {
					newTag = "p"
				}
				cell = setNodeTag(cell, newTag)
				replaceNode(table, cell)
			}
		}
	}
}

func (p *parser) cleanStyles(e *html.Node) {
	if e == nil || (e.Type == html.ElementNode && strings.ToLower(e.Data) == "svg") {
		return
	}

	for _, attr := range presentationalAttrs {
		removeAttr(e, attr)
	}

	if deprecatedSizeAttrElems[strings.ToLower(tagName(e))] {
		removeAttr(e, "width")
		removeAttr(e, "height")
	}

	for cur := firstElementChild(e); cur != nil; cur = nextElementSibling(cur) {
		p.cleanStyles(cur)
	}
}

func (p *parser) markDataTables(root *html.Node) {
	tables := getElementsByTagName(root, "table")
	for _, table := range tables {
		role := getAttr(table, "role")
		if role == "presentation" {
			p.dataTableFlags[table] = false
			continue
		}
		datatable := getAttr(table, "datatable")
		if datatable == "0" {
			p.dataTableFlags[table] = false
			continue
		}
		summary := getAttr(table, "summary")
		if summary != "" {
			p.dataTableFlags[table] = true
			continue
		}

		captions := getElementsByTagName(table, "caption")
		if len(captions) > 0 && captions[0].FirstChild != nil {
			p.dataTableFlags[table] = true
			continue
		}

		dataTableDescendants := []string{"col", "colgroup", "tfoot", "thead", "th"}
		isData := false
		for _, tag := range dataTableDescendants {
			if len(getElementsByTagName(table, tag)) > 0 {
				isData = true
				break
			}
		}
		if isData {
			p.dataTableFlags[table] = true
			continue
		}

		if len(getElementsByTagName(table, "table")) > 0 {
			p.dataTableFlags[table] = false
			continue
		}

		rows, columns := getRowAndColumnCount(table)
		if columns == 1 || rows == 1 {
			p.dataTableFlags[table] = false
			continue
		}
		if rows >= 10 || columns > 4 {
			p.dataTableFlags[table] = true
			continue
		}
		p.dataTableFlags[table] = rows*columns > 10
	}
}

func getRowAndColumnCount(table *html.Node) (int, int) {
	rows := 0
	columns := 0
	trs := getElementsByTagName(table, "tr")
	for _, tr := range trs {
		rowspanStr := getAttr(tr, "rowspan")
		rowspan := 0
		if rowspanStr != "" {
			rowspan, _ = strconv.Atoi(rowspanStr)
		}
		if rowspan == 0 {
			rowspan = 1
		}
		rows += rowspan

		columnsInThisRow := 0
		cells := getElementsByTagName(tr, "td")
		for _, cell := range cells {
			colspanStr := getAttr(cell, "colspan")
			colspan := 0
			if colspanStr != "" {
				colspan, _ = strconv.Atoi(colspanStr)
			}
			if colspan == 0 {
				colspan = 1
			}
			columnsInThisRow += colspan
		}
		if columnsInThisRow > columns {
			columns = columnsInThisRow
		}
	}
	return rows, columns
}

func (p *parser) fixLazyImages(root *html.Node) {
	nodes := getAllNodesWithTag(root, []string{"img", "picture", "figure"})
	for _, elem := range nodes {
		src := getAttr(elem, "src")
		if src != "" && rxB64DataURL.MatchString(src) {
			parts := rxB64DataURL.FindStringSubmatch(src)
			if len(parts) > 1 && parts[1] == "image/svg+xml" {
				continue
			}

			srcCouldBeRemoved := false
			for _, attr := range elem.Attr {
				if attr.Key == "src" {
					continue
				}
				if rxImageExtension.MatchString(attr.Val) {
					srcCouldBeRemoved = true
					break
				}
			}

			if srcCouldBeRemoved {
				b64starts := len(parts[0])
				b64length := len(src) - b64starts
				if b64length < 133 {
					removeAttr(elem, "src")
				}
			}
		}

		srcset := getAttr(elem, "srcset")
		hasSrc := src != ""
		hasSrcset := srcset != "" && srcset != "null"
		cls := strings.ToLower(className(elem))

		if (hasSrc || hasSrcset) && !strings.Contains(cls, "lazy") {
			continue
		}

		for _, attr := range elem.Attr {
			if attr.Key == "src" || attr.Key == "srcset" || attr.Key == "alt" {
				continue
			}
			var copyTo string
			if rxSrcsetCandidate.MatchString(attr.Val) {
				copyTo = "srcset"
			} else if rxSrcCandidate.MatchString(attr.Val) {
				copyTo = "src"
			}
			if copyTo != "" {
				tag := tagName(elem)
				if tag == "IMG" || tag == "PICTURE" {
					setAttr(elem, copyTo, attr.Val)
				} else if tag == "FIGURE" && len(getAllNodesWithTag(elem, []string{"img", "picture"})) == 0 {
					img := createElement("img")
					setAttr(img, copyTo, attr.Val)
					appendChild(elem, img)
				}
			}
		}
	}
}

func (p *parser) clean(e *html.Node, tag string) {
	isEmbed := tag == "object" || tag == "embed" || tag == "iframe"

	removeNodes(getAllNodesWithTag(e, []string{tag}), func(element *html.Node) bool {
		if isEmbed {
			for _, attr := range element.Attr {
				if p.allowedVideoRegex.MatchString(attr.Val) {
					return false
				}
			}
			if element.Data == "object" && p.allowedVideoRegex.MatchString(innerHTML(element)) {
				return false
			}
		}
		return true
	})
}

func (p *parser) cleanConditionally(e *html.Node, tag string) {
	if !p.flagIsActive(flagCleanConditionally) {
		return
	}

	removeNodes(getAllNodesWithTag(e, []string{tag}), func(node *html.Node) bool {
		isDataTable := func(t *html.Node) bool {
			v, ok := p.dataTableFlags[t]
			return ok && v
		}

		isList := tag == "ul" || tag == "ol"
		if !isList {
			listLength := 0
			listNodes := getAllNodesWithTag(node, []string{"ul", "ol"})
			for _, list := range listNodes {
				listLength += len(getInnerText(list, true))
			}
			nodeText := getInnerText(node, true)
			if len(nodeText) > 0 {
				isList = float64(listLength)/float64(len(nodeText)) > 0.9
			}
		}

		if tag == "table" && isDataTable(node) {
			return false
		}

		if hasAncestorTag(node, "table", -1, isDataTable) {
			return false
		}

		if hasAncestorTag(node, "code", 0, nil) {
			return false
		}

		tables := getElementsByTagName(node, "table")
		for _, tbl := range tables {
			if isDataTable(tbl) {
				return false
			}
		}

		weight := p.getClassWeight(node)

		contentScore := float64(0)
		if float64(weight)+contentScore < 0 {
			return true
		}

		if p.getCharCount(node, ",") < 10 {
			pCount := len(getElementsByTagName(node, "p"))
			imgCount := len(getElementsByTagName(node, "img"))
			liCount := len(getElementsByTagName(node, "li")) - 100
			inputCount := len(getElementsByTagName(node, "input"))
			headingDensity := p.getTextDensity(node, []string{"h1", "h2", "h3", "h4", "h5", "h6"})

			embedCount := 0
			embeds := getAllNodesWithTag(node, []string{"object", "embed", "iframe"})
			for _, embed := range embeds {
				for _, attr := range embed.Attr {
					if p.allowedVideoRegex.MatchString(attr.Val) {
						return false
					}
				}
				if embed.Data == "object" && p.allowedVideoRegex.MatchString(innerHTML(embed)) {
					return false
				}
				embedCount++
			}

			innerText := getInnerText(node, true)

			if rxAdWords.MatchString(innerText) || rxLoadingWords.MatchString(innerText) {
				return true
			}

			contentLength := len(innerText)
			linkDensity := p.getLinkDensity(node)

			textishTags := []string{"span", "li", "td", "blockquote", "dl", "div", "img", "ol", "p", "pre", "table", "ul"}
			textDensity := p.getTextDensity(node, textishTags)
			isFigureChild := hasAncestorTag(node, "figure", 0, nil)

			shouldRemove := false
			if !isFigureChild && imgCount > 1 && float64(pCount)/float64(imgCount) < 0.5 {
				shouldRemove = true
			}
			if !isList && liCount > pCount {
				shouldRemove = true
			}
			if inputCount > pCount/3 {
				shouldRemove = true
			}
			if !isList && !isFigureChild && headingDensity < 0.9 && contentLength < 25 && (imgCount == 0 || imgCount > 2) && linkDensity > 0 {
				shouldRemove = true
			}
			if !isList && weight < 25 && linkDensity > 0.2+p.linkDensityModifier {
				shouldRemove = true
			}
			if weight >= 25 && linkDensity > 0.5+p.linkDensityModifier {
				shouldRemove = true
			}
			if (embedCount == 1 && contentLength < 75) || embedCount > 1 {
				shouldRemove = true
			}
			if imgCount == 0 && textDensity == 0 {
				shouldRemove = true
			}

			if isList && shouldRemove {
				ch := children(node)
				for _, child := range ch {
					if len(children(child)) > 1 {
						return shouldRemove
					}
				}
				liTotal := len(getElementsByTagName(node, "li"))
				if imgCount == liTotal {
					return false
				}
			}

			return shouldRemove
		}
		return false
	})
}

func (p *parser) cleanMatchedNodes(e *html.Node, filter func(*html.Node, string) bool) {
	endOfSearchMarkerNode := getNextNode(e, true)
	next := getNextNode(e, false)
	for next != nil && next != endOfSearchMarkerNode {
		matchString := className(next) + " " + nodeID(next)
		if filter(next, matchString) {
			next = removeAndGetNext(next)
		} else {
			next = getNextNode(next, false)
		}
	}
}

func (p *parser) cleanHeaders(e *html.Node) {
	headingNodes := getAllNodesWithTag(e, []string{"h1", "h2"})
	removeNodes(headingNodes, func(node *html.Node) bool {
		return p.getClassWeight(node) < 0
	})
}
