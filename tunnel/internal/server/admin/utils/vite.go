package utils

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sync"
)

type ViteManifest struct {
	IndexHTML struct {
		File string   `json:"file"`
		CSS  []string `json:"css"`
	} `json:"index.html"`
}

var (
	viteTags template.HTML
	once     sync.Once
)

// GenerateViteTags is kept for backward compatibility with existing callers.
// It returns an empty string by default. Prefer using GenerateViteTagsFromBytes
// which accepts manifest bytes (for example, read from an embedded staticFS)
// and returns the proper tags.
func GenerateViteTags() template.HTML {
	return ""
}

// GenerateViteTagsFromBytes takes the raw bytes of a Vite manifest (manifest.json)
// and returns the HTML tags for CSS and JS. The result is memoized on first call.
func GenerateViteTagsFromBytes(manifestBytes []byte) template.HTML {
	once.Do(func() {
		viteTags = template.HTML(generateViteTagsInternal(manifestBytes))
	})
	return viteTags
}

func generateViteTagsInternal(manifestBytes []byte) string {
	// If manifest is empty, return empty tags.
	if len(manifestBytes) == 0 {
		return ""
	}

	var manifest ViteManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
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
