// Package web embeds static assets for the DeskSlice UI.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var embeddedFS embed.FS

// StaticFS returns the embedded static asset filesystem.
func StaticFS() (fs.FS, error) {
	return fs.Sub(embeddedFS, "static")
}
