package readability

var defaultTagsToScore = map[string]bool{
	"section": true,
	"h2":      true,
	"h3":      true,
	"h4":      true,
	"h5":      true,
	"h6":      true,
	"p":       true,
	"td":      true,
	"pre":     true,
}

var divToP = map[string]bool{
	"blockquote": true,
	"dl":         true,
	"div":        true,
	"img":        true,
	"ol":         true,
	"p":          true,
	"pre":        true,
	"table":      true,
	"ul":         true,
}

var alterToDivExceptions = map[string]bool{
	"div":     true,
	"article": true,
	"section": true,
	"p":       true,
	"ol":      true,
	"ul":      true,
}

var phrasingElems = map[string]bool{
	"abbr":     true,
	"audio":    true,
	"b":        true,
	"bdo":      true,
	"br":       true,
	"button":   true,
	"cite":     true,
	"code":     true,
	"data":     true,
	"datalist": true,
	"dfn":      true,
	"em":       true,
	"embed":    true,
	"i":        true,
	"img":      true,
	"input":    true,
	"kbd":      true,
	"label":    true,
	"mark":     true,
	"math":     true,
	"meter":    true,
	"noscript": true,
	"object":   true,
	"output":   true,
	"progress": true,
	"q":        true,
	"ruby":     true,
	"samp":     true,
	"script":   true,
	"select":   true,
	"small":    true,
	"span":     true,
	"strong":   true,
	"sub":      true,
	"sup":      true,
	"textarea": true,
	"time":     true,
	"var":      true,
	"wbr":      true,
}

var unlikelyRoles = map[string]bool{
	"menu":          true,
	"menubar":       true,
	"complementary": true,
	"navigation":    true,
	"alert":         true,
	"alertdialog":   true,
	"dialog":        true,
}

var presentationalAttrs = []string{
	"align",
	"background",
	"bgcolor",
	"border",
	"cellpadding",
	"cellspacing",
	"frame",
	"hspace",
	"rules",
	"style",
	"valign",
	"vspace",
}

var deprecatedSizeAttrElems = map[string]bool{
	"table": true,
	"th":    true,
	"td":    true,
	"hr":    true,
	"pre":   true,
}
