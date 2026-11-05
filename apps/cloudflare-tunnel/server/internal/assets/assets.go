// Package assets embeds the built single page application. The root build
// replaces internal/assets/dist with web/dist before compiling.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist returns the embedded UI rooted at the dist directory.
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// embed guarantees the directory exists; a failure here is a bug.
		panic(err)
	}
	return sub
}

// ReadFile reads a file from the embedded UI.
func ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(Dist(), name)
}
