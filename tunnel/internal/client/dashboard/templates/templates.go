//go:build !nodashboard

package templates

import "embed"

//go:embed index.html
var IndexTemplate embed.FS
