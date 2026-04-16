package cscore

import (
	"bytes"

	chromaHTML "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// RenderMarkdownToHTML converts markdown to HTML using goldmark with Chroma
// syntax highlighting (monokai style, inline CSS). GFM extensions enabled.
func RenderMarkdownToHTML(md string) string {
	if err := validateMarkdown(md); err != nil {
		return errorJSON(err)
	}

	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					chromaHTML.WithClasses(false), // inline style="" on each span
				),
			),
		),
	)

	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return errorJSON(err)
	}

	return jsonMarshal(map[string]string{"html": buf.String()})
}
