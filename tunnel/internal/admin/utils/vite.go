package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type ViteManifest struct {
	IndexHTML struct {
		File string   `json:"file"`
		CSS  []string `json:"css"`
	} `json:"index.html"`
}

var (
	viteTags string
	once     sync.Once
)

func GenerateViteTags() string {
	once.Do(func() {
		viteTags = generateViteTagsInternal()
	})
	return viteTags
}

func generateViteTagsInternal() string {
	manifestPath := findManifestPath()
	if manifestPath == "" {
		return ""
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ""
	}

	var manifest ViteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ""
	}

	var tags string

	for _, cssFile := range manifest.IndexHTML.CSS {
		tags += fmt.Sprintf(`<link rel="stylesheet" crossorigin href="/static/%s">`, cssFile)
	}

	if manifest.IndexHTML.File != "" {
		tags += fmt.Sprintf(`<script type="module" crossorigin src="/static/%s"></script>`, manifest.IndexHTML.File)
	}

	return tags
}

func findManifestPath() string {
	manifestPath := "web/dist/static/.vite/manifest.json"

	if _, err := os.Stat(manifestPath); err == nil {
		return manifestPath
	}

	return ""
}
