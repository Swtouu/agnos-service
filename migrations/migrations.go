// Package migrations embeds the .sql migration files into the compiled
// binary so cmd/api and cmd/seed can apply them at startup without depending
// on a separate init container or the files existing on disk at runtime —
// needed for platforms like Railway that don't have a distinct migration step.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
