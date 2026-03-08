// Package omnidex is the root package that embeds static web assets.
package omnidex

import "embed"

// StaticFiles holds all files under the static/ directory tree.
// Using embed.FS ensures the assets are always available at runtime, regardless
// of the working directory, eliminating 404 errors caused by filesystem path mismatches
// in containerised deployments.
//
//go:embed static
var StaticFiles embed.FS
