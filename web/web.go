// Package web exposes the embedded HTML templates and static assets so the
// server package can mount them without depending on the working directory.
package web

import "embed"

//go:embed templates/*.html
var Templates embed.FS

//go:embed static
var Static embed.FS
