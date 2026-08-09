package clientconfig

import (
	"strings"
	"testing"
)

func TestValidateTemplateAcceptsTunnelsAndGroups(t *testing.T) {
	tests := map[string]string{
		"empty":         "",
		"whitespace":    "   \n\n",
		"comments only": "# nothing here yet\n",
		"tunnels only": `tunnels:
  - name: web
    subdomain: acme-web
    port: 3000
`,
		"tunnels and groups": `tunnels:
  - name: web
    subdomain: acme-web
    port: 3000
  - name: api
    subdomain: acme-api
    port: 8000
groups:
  frontend: [web, api]
`,
		"stub tunnel with inline template": `tunnels:
  - name: payments
    type: stub
    subdomain: acme-payments
    response_format: json
    response_tmpl: '{"ok": true}'
`,
	}

	for name, template := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateTemplate(template); err != nil {
				t.Fatalf("expected valid template, got %v", err)
			}
		})
	}
}

func TestValidateTemplateRejects(t *testing.T) {
	tests := []struct {
		name     string
		template string
		contains string
	}{
		{
			name: "connection setting",
			template: `server_url: evil.example.com
tunnels:
  - name: web
    port: 3000
`,
			contains: `"server_url" is not allowed`,
		},
		{
			name: "machine local setting",
			template: `dashboard_port: 9999
`,
			contains: `"dashboard_port" is not allowed`,
		},
		{
			name:     "sequence at the top level",
			template: "- name: web\n  port: 3000\n",
			contains: "YAML mapping",
		},
		{
			name: "multiple documents",
			template: `tunnels:
  - name: web
    port: 3000
---
groups:
  frontend: [web]
`,
			contains: "single YAML document",
		},
		{
			name: "unnamed tunnel",
			template: `tunnels:
  - subdomain: acme-web
    port: 3000
`,
			contains: "needs a name",
		},
		{
			name: "duplicate tunnel names",
			template: `tunnels:
  - name: web
    port: 3000
  - name: web
    port: 3001
`,
			contains: `duplicate tunnel name "web"`,
		},
		{
			name: "stub template file",
			template: `tunnels:
  - name: payments
    type: stub
    subdomain: acme-payments
    response_format: json
    response_tmpl_file: ./stub.json
`,
			contains: "response_tmpl_file",
		},
		{
			name: "static tunnel",
			template: `tunnels:
  - name: site
    type: static
    subdomain: acme-site
    dir: ./public
`,
			contains: "static tunnels serve a directory on your machine",
		},
		{
			name: "group referencing unknown tunnel",
			template: `tunnels:
  - name: web
    port: 3000
groups:
  frontend: [web, api]
`,
			contains: `references unknown tunnel "api"`,
		},
		{
			name: "invalid subdomain",
			template: `tunnels:
  - name: web
    subdomain: acme_web
    port: 3000
`,
			contains: "subdomain",
		},
		{
			name:     "malformed yaml",
			template: "tunnels:\n  - name: web\n   port: 3000\n",
			contains: "yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTemplate(test.template)
			if err == nil {
				t.Fatal("expected template to be rejected")
			}
			if !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("expected error to mention %q, got %v", test.contains, err)
			}
		})
	}
}
