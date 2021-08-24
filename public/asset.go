package public

import (
	"embed"
	_ "embed"
)

//go:embed index.html
var IndexHTML string

//go:embed index.js
var IndexJS string

//go:embed css
var CssFiles embed.FS

//go:embed js
var JsFiles embed.FS
