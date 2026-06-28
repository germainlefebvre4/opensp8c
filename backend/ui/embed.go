package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embedFS embed.FS

// FS returns the frontend dist as a sub-filesystem rooted at dist/.
func FS() fs.FS {
	sub, _ := fs.Sub(embedFS, "dist")
	return sub
}
