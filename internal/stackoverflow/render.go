package stackoverflow

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// ToMarkdown returns the rendered Markdown view of a Result, ready for the
// existing render.Markdown() glamour pipeline. Includes the required
// "Powered by Stack Exchange" attribution per the API ToS.
func ToMarkdown(r *Result, query string) string {
	if r == nil || len(r.Questions) == 0 {
		return fmt.Sprintf("# Stack Overflow: %s\n\n*No results.*\n", query)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Stack Overflow: %s\n\n", query)
	fmt.Fprintf(&b, "*%d result(s).*\n\n", len(r.Questions))

	for i, q := range r.Questions {
		marker := "✗"
		if q.IsAnswered {
			marker = "✓"
		}
		fmt.Fprintf(&b, "## %d. %s\n\n", i+1, html.UnescapeString(q.Title))
		fmt.Fprintf(&b, "**%s** · %d↑ · %d answer(s) · ", marker, q.Score, q.AnswerCount)
		if q.CreationDate > 0 {
			fmt.Fprintf(&b, "%s · ", time.Unix(q.CreationDate, 0).UTC().Format("2006-01-02"))
		}
		if len(q.Tags) > 0 {
			fmt.Fprintf(&b, "tags: `%s`\n\n", strings.Join(q.Tags, "`, `"))
		} else {
			b.WriteString("\n\n")
		}
		body := stripHTML(q.Body)
		if len(body) > 800 {
			body = body[:800] + "…"
		}
		if body != "" {
			b.WriteString(body)
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "→ <%s>\n\n", q.Link)
		b.WriteString("---\n\n")
	}

	fmt.Fprintf(&b,
		"*Powered by Stack Exchange — content licensed CC BY-SA 4.0. "+
			"Quota remaining: %d / %d.*\n",
		r.QuotaRemaining, r.QuotaMax)
	if r.Backoff > 0 {
		fmt.Fprintf(&b,
			"\n*The API requested a %ds backoff — vör's 24h cache "+
				"absorbs this for repeat queries.*\n", r.Backoff)
	}
	return b.String()
}

// stripHTML is a deliberately small HTML-to-text reducer. We don't ship a full
// HTML parser — Stack Overflow body fragments are well-formed enough that
// regex-based unwrapping is fine for terminal display.
var (
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	wsRe    = regexp.MustCompile(`\n{3,}`)
	preRe   = regexp.MustCompile(`(?s)<pre>(.*?)</pre>`)
	codeRe  = regexp.MustCompile(`<code>(.*?)</code>`)
	brRe    = regexp.MustCompile(`(?i)<br\s*/?>`)
	pCloseRe = regexp.MustCompile(`(?i)</p>`)
)

func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	// preserve <pre> blocks as fenced code
	s = preRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := preRe.FindStringSubmatch(m)[1]
		inner = tagRe.ReplaceAllString(inner, "")
		inner = html.UnescapeString(inner)
		return "\n```\n" + strings.TrimSpace(inner) + "\n```\n"
	})
	// inline <code> as backticks
	s = codeRe.ReplaceAllString(s, "`$1`")
	// <br> and </p> become newlines
	s = brRe.ReplaceAllString(s, "\n")
	s = pCloseRe.ReplaceAllString(s, "\n\n")
	// drop everything else
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = wsRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
