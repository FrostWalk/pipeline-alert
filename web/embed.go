package web

import "embed"

// StaticFiles holds the compiled frontend assets from the Vite build.
// The dist/ directory must be populated by running `pnpm build` before `go build`.
//
//go:embed all:dist
var StaticFiles embed.FS
