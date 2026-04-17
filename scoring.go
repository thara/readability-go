package readability

import (
	"math"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

type contentScore struct {
	score float64
}

func (p *parser) initializeNode(node *html.Node) {
	cs := &contentScore{score: 0}
	switch tagName(node) {
	case "DIV":
		cs.score += 5
	case "PRE", "TD", "BLOCKQUOTE":
		cs.score += 3
	case "ADDRESS", "OL", "UL", "DL", "DD", "DT", "LI", "FORM":
		cs.score -= 3
	case "H1", "H2", "H3", "H4", "H5", "H6", "TH":
		cs.score -= 5
	}
	cs.score += float64(p.getClassWeight(node))
	p.scores[node] = cs
}

func (p *parser) getClassWeight(node *html.Node) int {
	if !p.flagIsActive(flagWeightClasses) {
		return 0
	}

	weight := 0

	cls := className(node)
	if cls != "" {
		if rxNegative.MatchString(cls) {
			weight -= 25
		}
		if rxPositive.MatchString(cls) {
			weight += 25
		}
	}

	id := nodeID(node)
	if id != "" {
		if rxNegative.MatchString(id) {
			weight -= 25
		}
		if rxPositive.MatchString(id) {
			weight += 25
		}
	}

	return weight
}

func (p *parser) getLinkDensity(element *html.Node) float64 {
	textLength := len(getInnerText(element, true))
	if textLength == 0 {
		return 0
	}

	linkLength := 0
	links := getElementsByTagName(element, "a")
	for _, link := range links {
		href := getAttr(link, "href")
		coefficient := 1.0
		if href != "" && rxHashURL.MatchString(href) {
			coefficient = 0.3
		}
		linkLength += int(float64(len(getInnerText(link, true))) * coefficient)
	}

	return float64(linkLength) / float64(textLength)
}

func (p *parser) getTextDensity(e *html.Node, tags []string) float64 {
	textLength := len(getInnerText(e, true))
	if textLength == 0 {
		return 0
	}
	childrenLength := 0
	ch := getAllNodesWithTag(e, tags)
	for _, child := range ch {
		childrenLength += len(getInnerText(child, true))
	}
	return float64(childrenLength) / float64(textLength)
}

func (p *parser) getCharCount(e *html.Node, sep string) int {
	if sep == "" {
		sep = ","
	}
	return strings.Count(getInnerText(e, true), sep)
}

func textSimilarity(textA, textB string) float64 {
	tokensA := filterEmpty(rxTokenize.Split(strings.ToLower(textA), -1))
	tokensB := filterEmpty(rxTokenize.Split(strings.ToLower(textB), -1))
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	tokensASet := make(map[string]bool)
	for _, t := range tokensA {
		tokensASet[t] = true
	}

	var uniqTokensB []string
	for _, t := range tokensB {
		if !tokensASet[t] {
			uniqTokensB = append(uniqTokensB, t)
		}
	}

	distanceB := float64(len(strings.Join(uniqTokensB, " "))) / float64(len(strings.Join(tokensB, " ")))
	return 1 - distanceB
}

func filterEmpty(ss []string) []string {
	result := ss[:0]
	for _, s := range ss {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func (p *parser) isValidByline(node *html.Node, matchString string) bool {
	rel := getAttr(node, "rel")
	itemprop := getAttr(node, "itemprop")
	bylineLength := len(strings.TrimSpace(textContent(node)))

	return (rel == "author" ||
		(itemprop != "" && strings.Contains(itemprop, "author")) ||
		rxByline.MatchString(matchString)) &&
		bylineLength > 0 &&
		bylineLength < 100
}

func (p *parser) headerDuplicatesTitle(node *html.Node) bool {
	tag := tagName(node)
	if tag != "H1" && tag != "H2" {
		return false
	}
	heading := getInnerText(node, false)
	return textSimilarity(p.articleTitle, heading) > 0.75
}

func isProbablyVisible(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return true
	}
	style := getAttr(node, "style")
	styleLower := strings.ToLower(strings.ReplaceAll(style, " ", ""))
	if strings.Contains(styleLower, "display:none") {
		return false
	}
	if strings.Contains(styleLower, "visibility:hidden") {
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

func (p *parser) grabArticle() *html.Node {
	body := documentBody(p.doc)
	if body == nil {
		return nil
	}

	for {
		stripUnlikelyCandidates := p.flagIsActive(flagStripUnlikelys)

		var elementsToScore []*html.Node
		node := documentElement(p.doc)
		shouldRemoveTitleHeader := true

		for node != nil {
			if tagName(node) == "HTML" {
				p.articleLang = getAttr(node, "lang")
			}

			matchString := className(node) + " " + nodeID(node)

			if !isProbablyVisible(node) {
				node = removeAndGetNext(node)
				continue
			}

			if getAttr(node, "aria-modal") == "true" && getAttr(node, "role") == "dialog" {
				node = removeAndGetNext(node)
				continue
			}

			if p.articleByline == "" && p.metadata.byline == "" && p.isValidByline(node, matchString) {
				endOfSearchMarkerNode := getNextNode(node, true)
				next := getNextNode(node, false)
				var itemPropNameNode *html.Node
				for next != nil && next != endOfSearchMarkerNode {
					itemprop := getAttr(next, "itemprop")
					if itemprop != "" && strings.Contains(itemprop, "name") {
						itemPropNameNode = next
						break
					}
					next = getNextNode(next, false)
				}
				if itemPropNameNode != nil {
					p.articleByline = strings.TrimSpace(textContent(itemPropNameNode))
				} else {
					p.articleByline = strings.TrimSpace(textContent(node))
				}
				node = removeAndGetNext(node)
				continue
			}

			if shouldRemoveTitleHeader && p.headerDuplicatesTitle(node) {
				shouldRemoveTitleHeader = false
				node = removeAndGetNext(node)
				continue
			}

			if stripUnlikelyCandidates {
				if rxUnlikelyCandidates.MatchString(matchString) &&
					!rxOkMaybeItsACandidate.MatchString(matchString) &&
					!hasAncestorTag(node, "table", 0, nil) &&
					!hasAncestorTag(node, "code", 0, nil) &&
					tagName(node) != "BODY" &&
					tagName(node) != "A" {
					node = removeAndGetNext(node)
					continue
				}

				role := getAttr(node, "role")
				if unlikelyRoles[role] {
					node = removeAndGetNext(node)
					continue
				}
			}

			tag := tagName(node)
			if (tag == "DIV" || tag == "SECTION" || tag == "HEADER" ||
				tag == "H1" || tag == "H2" || tag == "H3" ||
				tag == "H4" || tag == "H5" || tag == "H6") &&
				isElementWithoutContent(node) {
				node = removeAndGetNext(node)
				continue
			}

			if defaultTagsToScore[strings.ToLower(tag)] {
				elementsToScore = append(elementsToScore, node)
			}

			if tag == "DIV" {
				childNode := node.FirstChild
				for childNode != nil {
					nextSibling := childNode.NextSibling
					if isPhrasingContent(childNode) {
						fragment := createElement("div")
						for childNode != nil && isPhrasingContent(childNode) {
							nextSibling = childNode.NextSibling
							appendChild(fragment, childNode)
							childNode = nextSibling
						}
						for fragment.FirstChild != nil && isWhitespace(fragment.FirstChild) {
							removeNode(fragment.FirstChild)
						}
						for fragment.LastChild != nil && isWhitespace(fragment.LastChild) {
							removeNode(fragment.LastChild)
						}
						if fragment.FirstChild != nil {
							newP := createElement("p")
							for fragment.FirstChild != nil {
								appendChild(newP, fragment.FirstChild)
							}
							insertBefore(node, newP, nextSibling)
						}
					}
					childNode = nextSibling
				}

				if hasSingleTagInsideElement(node, "P") && p.getLinkDensity(node) < 0.25 {
					newNode := children(node)[0]
					replaceNode(node, newNode)
					node = newNode
					elementsToScore = append(elementsToScore, node)
				} else if !hasChildBlockElement(node) {
					node = setNodeTag(node, "p")
					elementsToScore = append(elementsToScore, node)
				}
			}
			node = getNextNode(node, false)
		}

		var candidates []*html.Node
		for _, elementToScore := range elementsToScore {
			if elementToScore.Parent == nil {
				continue
			}
			if elementToScore.Parent.Type != html.ElementNode {
				continue
			}

			innerText := getInnerText(elementToScore, true)
			if len(innerText) < 25 {
				continue
			}

			ancestors := getNodeAncestors(elementToScore, 5)
			if len(ancestors) == 0 {
				continue
			}

			cs := float64(1)
			cs += float64(len(rxCommas.FindAllString(innerText, -1))) + 1
			cs += math.Min(math.Floor(float64(len(innerText))/100), 3)

			for level, ancestor := range ancestors {
				if ancestor.Type != html.ElementNode {
					continue
				}
				if ancestor.Parent == nil || ancestor.Parent.Type != html.ElementNode {
					continue
				}

				if p.scores[ancestor] == nil {
					p.initializeNode(ancestor)
					candidates = append(candidates, ancestor)
				}

				var scoreDivider float64
				switch level {
				case 0:
					scoreDivider = 1
				case 1:
					scoreDivider = 2
				default:
					scoreDivider = float64(level) * 3
				}
				p.scores[ancestor].score += cs / scoreDivider
			}
		}

		var topCandidates []*html.Node
		for _, candidate := range candidates {
			sc := p.scores[candidate]
			if sc == nil {
				continue
			}
			candidateScore := sc.score * (1 - p.getLinkDensity(candidate))
			sc.score = candidateScore

			for t := 0; t < p.nbTopCandidates; t++ {
				if t >= len(topCandidates) {
					topCandidates = append(topCandidates, candidate)
					break
				}
				aTopScore := p.scores[topCandidates[t]]
				if aTopScore == nil || candidateScore > aTopScore.score {
					// Insert at position t
					topCandidates = append(topCandidates, nil)
					copy(topCandidates[t+1:], topCandidates[t:])
					topCandidates[t] = candidate
					if len(topCandidates) > p.nbTopCandidates {
						topCandidates = topCandidates[:p.nbTopCandidates]
					}
					break
				}
			}
		}

		var topCandidate *html.Node
		if len(topCandidates) > 0 {
			topCandidate = topCandidates[0]
		}
		neededToCreateTopCandidate := false
		var parentOfTopCandidate *html.Node

		if topCandidate == nil || tagName(topCandidate) == "BODY" {
			topCandidate = createElement("div")
			neededToCreateTopCandidate = true
			for body.FirstChild != nil {
				appendChild(topCandidate, body.FirstChild)
			}
			appendChild(body, topCandidate)
			p.initializeNode(topCandidate)
		} else {
			var alternativeCandidateAncestors [][]*html.Node
			for i := 1; i < len(topCandidates); i++ {
				tcScore := p.scores[topCandidate]
				altScore := p.scores[topCandidates[i]]
				if tcScore != nil && altScore != nil && altScore.score/tcScore.score >= 0.75 {
					alternativeCandidateAncestors = append(alternativeCandidateAncestors, getNodeAncestors(topCandidates[i], 0))
				}
			}

			minimumTopCandidates := 3
			if len(alternativeCandidateAncestors) >= minimumTopCandidates {
				parentOfTopCandidate = topCandidate.Parent
				for parentOfTopCandidate != nil && tagName(parentOfTopCandidate) != "BODY" {
					listsContainingThisAncestor := 0
					for ancestorIdx := 0; ancestorIdx < len(alternativeCandidateAncestors) && listsContainingThisAncestor < minimumTopCandidates; ancestorIdx++ {
						if slices.Contains(alternativeCandidateAncestors[ancestorIdx], parentOfTopCandidate) {
							listsContainingThisAncestor++
						}
					}
					if listsContainingThisAncestor >= minimumTopCandidates {
						topCandidate = parentOfTopCandidate
						break
					}
					parentOfTopCandidate = parentOfTopCandidate.Parent
				}
			}

			if p.scores[topCandidate] == nil {
				p.initializeNode(topCandidate)
			}

			parentOfTopCandidate = topCandidate.Parent
			lastScore := p.scores[topCandidate].score
			scoreThreshold := lastScore / 3
			for parentOfTopCandidate != nil && tagName(parentOfTopCandidate) != "BODY" {
				parentScore := p.scores[parentOfTopCandidate]
				if parentScore == nil {
					parentOfTopCandidate = parentOfTopCandidate.Parent
					continue
				}
				if parentScore.score < scoreThreshold {
					break
				}
				if parentScore.score > lastScore {
					topCandidate = parentOfTopCandidate
					break
				}
				lastScore = parentScore.score
				parentOfTopCandidate = parentOfTopCandidate.Parent
			}

			parentOfTopCandidate = topCandidate.Parent
			for parentOfTopCandidate != nil && tagName(parentOfTopCandidate) != "BODY" && len(children(parentOfTopCandidate)) == 1 {
				topCandidate = parentOfTopCandidate
				parentOfTopCandidate = topCandidate.Parent
			}
			if p.scores[topCandidate] == nil {
				p.initializeNode(topCandidate)
			}
		}

		articleContent := createElement("div")

		siblingScoreThreshold := math.Max(10, p.scores[topCandidate].score*0.2)
		parentOfTopCandidate = topCandidate.Parent
		siblings := children(parentOfTopCandidate)

		for s := 0; s < len(siblings); s++ {
			sibling := siblings[s]
			appendIt := false

			if sibling == topCandidate {
				appendIt = true
			} else {
				contentBonus := float64(0)
				if className(sibling) == className(topCandidate) && className(topCandidate) != "" {
					contentBonus += p.scores[topCandidate].score * 0.2
				}

				sibScore := p.scores[sibling]
				if sibScore != nil && sibScore.score+contentBonus >= siblingScoreThreshold {
					appendIt = true
				} else if tagName(sibling) == "P" {
					linkDensity := p.getLinkDensity(sibling)
					nodeContent := getInnerText(sibling, true)
					nodeLength := len(nodeContent)

					if nodeLength > 80 && linkDensity < 0.25 {
						appendIt = true
					} else if nodeLength < 80 && nodeLength > 0 && linkDensity == 0 && rxSentenceEnd.MatchString(nodeContent) {
						appendIt = true
					}
				}
			}

			if appendIt {
				if !alterToDivExceptions[strings.ToLower(tagName(sibling))] {
					setNodeTag(sibling, "div")
				}

				appendChild(articleContent, sibling)
				siblings = children(parentOfTopCandidate)
				s--
			}
		}

		p.prepArticle(articleContent)

		if neededToCreateTopCandidate {
			setAttr(topCandidate, "id", "readability-page-1")
			setAttr(topCandidate, "class", "page")
		} else {
			div := createElement("div")
			setAttr(div, "id", "readability-page-1")
			setAttr(div, "class", "page")
			for articleContent.FirstChild != nil {
				appendChild(div, articleContent.FirstChild)
			}
			appendChild(articleContent, div)
		}

		parseSuccessful := true

		textLength := len(getInnerText(articleContent, true))
		if textLength < p.charThreshold {
			parseSuccessful = false

			p.attempts = append(p.attempts, parseAttempt{
				articleContent: articleContent,
				textLength:     textLength,
			})

			if p.flagIsActive(flagStripUnlikelys) {
				p.removeFlag(flagStripUnlikelys)
			} else if p.flagIsActive(flagWeightClasses) {
				p.removeFlag(flagWeightClasses)
			} else if p.flagIsActive(flagCleanConditionally) {
				p.removeFlag(flagCleanConditionally)
			} else {
				// Sort attempts by text length descending
				best := 0
				for i := 1; i < len(p.attempts); i++ {
					if p.attempts[i].textLength > p.attempts[best].textLength {
						best = i
					}
				}
				if p.attempts[best].textLength == 0 {
					return nil
				}
				articleContent = p.attempts[best].articleContent
				parseSuccessful = true
			}
		}

		if parseSuccessful {
			ancestors := []*html.Node{parentOfTopCandidate, topCandidate}
			ancestors = append(ancestors, getNodeAncestors(parentOfTopCandidate, 0)...)
			for _, ancestor := range ancestors {
				if ancestor == nil || ancestor.Type != html.ElementNode {
					continue
				}
				articleDir := getAttr(ancestor, "dir")
				if articleDir != "" {
					p.articleDir = articleDir
					break
				}
			}
			return articleContent
		}

		// Re-parse for retry
		p.reparseDoc()
		p.removeScripts()
		p.prepDocument()
		p.articleByline = ""
		body = documentBody(p.doc)
		if body == nil {
			return nil
		}
	}
}
