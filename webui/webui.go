// Package webui embeds the test console HTML so cmd/api can serve it
// directly — no separate static server needed to reach it once deployed.
package webui

import "embed"

//go:embed index.html
var FS embed.FS
