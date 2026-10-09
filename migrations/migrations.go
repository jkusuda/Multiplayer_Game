// Package migrations bundles the .sql files in this folder into the compiled server.
// See 0000_how_migrations_work.sql for how to add one.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
