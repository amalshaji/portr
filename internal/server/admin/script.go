package admin

import (
	"encoding/json"
	"log"
	"os"
)

const viteDistDir = "./internal/server/admin/web/dist"

type manifest struct {
	IndexHTML struct {
		CSS     []string `json:"css"`
		File    string   `json:"file"`
		IsEntry bool     `json:"isEntry"`
		Src     string   `json:"src"`
	} `json:"index.html"`
}

func getViteTags() string {
	manifestFileContents, err := os.ReadFile(viteDistDir + "/static/.vite/manifest.json")
	if err != nil {
		log.Fatal(err)
	}

	var manifest manifest
	if err := json.Unmarshal(manifestFileContents, &manifest); err != nil {
		log.Fatal(err)
	}

	var tags string

	csses := manifest.IndexHTML.CSS
	if len(csses) > 0 {
		for _, css := range csses {
			tags += "<link rel=\"stylesheet\" crossorigin href=\"/static/" + css + "\">"
		}
	}

	file := manifest.IndexHTML.File
	if file != "" {
		tags += "<script type=\"module\" crossorigin src=\"/static/" + file + "\"></script>"
	}

	return tags
}
