// Package webapp embeds the built Next.js UI bundle so it ships inside any
// binary that imports the frontend server — the OSS UI and out-of-tree builds
// (notably the enterprise UI) alike. Importers no longer carry their own
// //go:embed of the bundle; they get it transitively from this package.
//
// The contents of out/ are produced by `yarn build` and are gitignored. A
// committed out/.keep keeps this package compilable before a build has run.
// Builds that need a real UI (the OSS Dockerfile, the enterprise Dockerfile
// via its go.mod replace onto a yarn-built OSS clone) populate out/ before
// `go build`, so the embed picks up the actual bundle.
package webapp

import (
	"embed"
	"io/fs"
)

//go:embed all:out
var bundle embed.FS

// FS returns the contents of the built webapp/out directory as a filesystem
// rooted at the bundle root (so "index.html" resolves directly).
func FS() (fs.FS, error) {
	return fs.Sub(bundle, "out")
}
