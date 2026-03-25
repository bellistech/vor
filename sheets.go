package cs

import "embed"

//go:embed sheets/*/*.md
var EmbeddedSheets embed.FS
