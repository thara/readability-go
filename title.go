package readability

import (
	"regexp"
	"strings"
)

func (p *parser) getArticleTitle() string {
	curTitle := ""
	origTitle := ""

	titles := getElementsByTagName(p.doc, "title")
	if len(titles) > 0 {
		curTitle = getInnerText(titles[0], true)
		origTitle = curTitle
	}

	titleHadHierarchicalSeparators := false
	wordCount := func(s string) int {
		fields := strings.Fields(s)
		return len(fields)
	}

	titleSeparators := `\|\-\x{2013}\x{2014}\\\/>\x{00BB}`

	rxSep := regexp.MustCompile(`\s[` + titleSeparators + `]\s`)
	if rxSep.MatchString(curTitle) {
		titleHadHierarchicalSeparators = rxTitleHierarchicalSep.MatchString(curTitle)

		allSeps := rxSep.FindAllStringIndex(origTitle, -1)
		if len(allSeps) > 0 {
			lastSep := allSeps[len(allSeps)-1]
			curTitle = origTitle[:lastSep[0]]
		}

		if wordCount(curTitle) < 3 {
			rxRemoveFirst := regexp.MustCompile(`^[^` + titleSeparators + `]*[` + titleSeparators + `]`)
			curTitle = rxRemoveFirst.ReplaceAllString(origTitle, "")
		}
	} else if strings.Contains(curTitle, ": ") {
		headings := getAllNodesWithTag(p.doc, []string{"h1", "h2"})
		trimmedTitle := strings.TrimSpace(curTitle)
		matched := false
		for _, heading := range headings {
			if strings.TrimSpace(textContent(heading)) == trimmedTitle {
				matched = true
				break
			}
		}

		if !matched {
			lastColon := strings.LastIndex(origTitle, ":")
			curTitle = origTitle[lastColon+1:]

			if wordCount(curTitle) < 3 {
				firstColon := strings.Index(origTitle, ":")
				curTitle = origTitle[firstColon+1:]
			} else if wordCount(origTitle[:strings.Index(origTitle, ":")]) > 5 {
				curTitle = origTitle
			}
		}
	} else if len(curTitle) > 150 || len(curTitle) < 15 {
		hOnes := getElementsByTagName(p.doc, "h1")
		if len(hOnes) == 1 {
			curTitle = getInnerText(hOnes[0], true)
		}
	}

	curTitle = strings.TrimSpace(rxNormalize.ReplaceAllString(curTitle, " "))

	curTitleWordCount := wordCount(curTitle)
	if curTitleWordCount <= 4 &&
		(!titleHadHierarchicalSeparators ||
			curTitleWordCount != wordCount(rxSep.ReplaceAllString(origTitle, ""))-1) {
		curTitle = origTitle
	}

	return curTitle
}
