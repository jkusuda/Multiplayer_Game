// Package migrations bundles the .sql files in this folder into the server.
// Each file runs once, in name order, on startup. To change the database, add
// the next numbered file; never edit one that has already run.
package migrations

import "embed"

// FS holds every .sql file in this folder.
//
//go:embed *.sql
var FS embed.FS
