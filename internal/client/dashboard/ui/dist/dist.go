package dist

import "embed"

//go:embed static/*
var EmbededDirStatic embed.FS
