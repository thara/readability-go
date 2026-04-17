package readability

import "regexp"

var (
	rxUnlikelyCandidates = regexp.MustCompile(`(?i)-ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote`)

	rxOkMaybeItsACandidate = regexp.MustCompile(`(?i)and|article|body|column|content|main|mathjax|shadow`)

	rxPositive = regexp.MustCompile(`(?i)article|body|content|entry|hentry|h-entry|main|page|pagination|post|text|blog|story`)

	rxNegative = regexp.MustCompile(`(?i)-ad-|hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|footer|gdpr|masthead|media|meta|outbrain|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|widget`)

	rxByline = regexp.MustCompile(`(?i)byline|author|dateline|writtenby|p-author`)

	rxNormalize = regexp.MustCompile(`\s{2,}`)

	rxVideos = regexp.MustCompile(`(?i)//(www\.)?((dailymotion|youtube|youtube-nocookie|player\.vimeo|v\.qq|bilibili|live\.bilibili)\.com|(archive|upload\.wikimedia)\.org|player\.twitch\.tv)`)

	rxShareElements = regexp.MustCompile(`(?i)(\b|_)(share|sharedaddy)(\b|_)`)

	rxTokenize = regexp.MustCompile(`\W+`)

	rxWhitespace = regexp.MustCompile(`^\s*$`)

	rxHasContent = regexp.MustCompile(`\S$`)

	rxHashURL = regexp.MustCompile(`^#.+`)

	rxSrcsetURL = regexp.MustCompile(`(\S+)(\s+[\d.]+[xw])?(\s*(?:,|$))`)

	rxB64DataURL = regexp.MustCompile(`(?i)^data:\s*([^\s;,]+)\s*;\s*base64\s*,`)

	rxCommas = regexp.MustCompile("[\u002C\u060C\uFE50\uFE10\uFE11\u2E41\u2E34\u2E32\uFF0C]")

	rxJSONLdArticleTypes = regexp.MustCompile(`^Article|AdvertiserContentArticle|NewsArticle|AnalysisNewsArticle|AskPublicNewsArticle|BackgroundNewsArticle|OpinionNewsArticle|ReportageNewsArticle|ReviewNewsArticle|Report|SatiricalArticle|ScholarlyArticle|MedicalScholarlyArticle|SocialMediaPosting|BlogPosting|LiveBlogPosting|DiscussionForumPosting|TechArticle|APIReference$`)

	rxAdWords = regexp.MustCompile(`(?i)^(ad(vertising|vertisement)?|pub(licité)?|werb(ung)?|广告|Реклама|Anuncio)$`)

	rxLoadingWords = regexp.MustCompile(`(?i)^((loading|正在加载|Загрузка|chargement|cargando)(…|\.\.\.)?)$`)

	rxImageExtension = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|webp)`)

	rxSrcsetCandidate = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|webp)\s+\d`)

	rxSrcCandidate = regexp.MustCompile(`(?i)^\s*\S+\.(jpg|jpeg|png|webp)\S*\s*$`)

	rxTitleHierarchicalSep = regexp.MustCompile(`\s[\\\/>\x{00BB}]\s`)

	rxPropertyPattern = regexp.MustCompile(`(?i)\s*(article|dc|dcterm|og|twitter)\s*:\s*(author|creator|description|published_time|title|site_name)\s*`)

	rxNamePattern = regexp.MustCompile(`(?i)^\s*(?:(dc|dcterm|og|twitter|parsely|weibo:(article|webpage))\s*[-\.:]?\s*)?(author|creator|pub-date|description|title|site_name)\s*$`)

	rxCDATA = regexp.MustCompile(`(?s)^\s*<!\[CDATA\[|\]\]>\s*$`)

	rxSchemaOrg = regexp.MustCompile(`^https?://schema\.org/?$`)

	rxSentenceEnd = regexp.MustCompile(`\.( |$)`)
)
