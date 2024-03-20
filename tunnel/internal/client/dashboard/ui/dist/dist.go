package dist

import "embed"

//go:embed static/*
var EmbeddedDirStatic embed.FS

//go:embed static/.vite/manifest.json
var ManifestString string
