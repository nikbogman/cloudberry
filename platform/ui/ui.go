// Package ui embeds the browser app's static files into the Edge
// binary. It lives here, beside the files themselves, because go:embed
// patterns are relative to their own package directory and cannot climb
// out of it -- cmd/edge-api can't reach up to ../../ui.
//
// The globs are deliberate: a new .html/.js/.css file is picked up with
// no edit here, and Go fails the build if any pattern matches nothing.
package ui

import "embed"

//go:embed *.html *.js *.css *.webmanifest icons
var Assets embed.FS
