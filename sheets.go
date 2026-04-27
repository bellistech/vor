package vor

import "embed"

//go:embed sheets/*/*.md
var EmbeddedSheets embed.FS

//go:embed detail/*/*.md
var EmbeddedDetails embed.FS
