package utils

import (
	_ "embed"

	"github.com/valyala/fasttemplate"
)

//go:embed error-templates/local-server-not-online.html
var LocalServerNotOnlineText string

func LocalServerNotOnline(endpoint string) string {
	return LocalServerNotOnlineText
}

//go:embed error-templates/unregistered-subdomain.html
var UnregisteredSubdomainText string

func UnregisteredSubdomain(subdomain string) string {
	t := fasttemplate.New(UnregisteredSubdomainText, "{{", "}}")
	return t.ExecuteString(map[string]any{"subdomain": subdomain})
}

//go:embed error-templates/connection-lost.html
var ConnectionLostText string

func ConnectionLost() string {
	return ConnectionLostText
}
