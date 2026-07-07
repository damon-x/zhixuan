package static

import "embed"

// Dist holds the embedded frontend build output (web/dist).
// Files are copied into ./dist at build time by deploy.sh.
//
//go:embed all:dist
var Dist embed.FS
