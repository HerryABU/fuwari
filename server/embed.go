package main

import "embed"

//go:embed all:dist
var distFS embed.FS

//go:embed all:assets
var assetsFS embed.FS
