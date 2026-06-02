// Package web embeds HTML templates and static assets into the binary.
package web

import "embed"

// TemplatesFS contains the HTML templates.
//
//go:embed templates/*.html
var TemplatesFS embed.FS

// StaticFS contains the static assets (CSS, etc).
//
//go:embed static/*
var StaticFS embed.FS
