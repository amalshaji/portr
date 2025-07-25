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

// GenerateViteTags generates HTML tags for production Vite assets
func GenerateViteTags() string {
	once.Do(func() {
		viteTags = generateViteTagsInternal()
	})
	return viteTags
}

func generateViteTagsInternal() string {
	// Find the manifest.json file
	manifestPath := findManifestPath()
	if manifestPath == "" {
		return ""
	}

	// Read manifest file
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ""
	}

	// Parse manifest
	var manifest ViteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ""
	}

	var tags string

	// Add CSS links
	for _, cssFile := range manifest.IndexHTML.CSS {
		tags += fmt.Sprintf(`<link rel="stylesheet" crossorigin href="/static/%s">`, cssFile)
	}

	// Add JS script
	if manifest.IndexHTML.File != "" {
		tags += fmt.Sprintf(`<script type="module" crossorigin src="/static/%s"></script>`, manifest.IndexHTML.File)
	}

	return tags
}

func findManifestPath() string {
	// Use only the web/dist/static/.vite/manifest.json path
	manifestPath := "web/dist/static/.vite/manifest.json"

	if _, err := os.Stat(manifestPath); err == nil {
		return manifestPath
	}

	return ""
}
