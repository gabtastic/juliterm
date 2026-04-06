package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web
var webFS embed.FS

// webFileSystem returns an http.FileSystem serving the embedded SPA.
// It strips the "web" prefix so files are served from root.
// Returns nil if the embedded directory is empty (dev mode).
func webFileSystem() http.FileSystem {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil
	}
	return http.FS(sub)
}
