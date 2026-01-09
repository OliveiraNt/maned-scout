// Package ui provides embedded static files for the web user interface.
// It contains the static assets (HTML, CSS, JavaScript, images, etc.) that are embedded
// into the binary at compile time using Go's embed directive.
package ui

import (
	"embed"
)

// StaticFiles is an embedded file system containing the files located under the "static" directory.
//
//go:embed static/*
var StaticFiles embed.FS
