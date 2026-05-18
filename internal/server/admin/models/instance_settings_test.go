package models

import "testing"

func TestNormalizeAllowedDomains(t *testing.T) {
	got := NormalizeAllowedDomains(" Example.com, @Example.com, api.example.com. , acme.co ")
	want := "example.com, api.example.com, acme.co"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEmailMatchesAllowedDomainsRequiresExactDomain(t *testing.T) {
	domains := "example.com, api.example.com"

	if !EmailMatchesAllowedDomains("user@example.com", domains) {
		t.Fatalf("expected example.com email to match")
	}

	if EmailMatchesAllowedDomains("user@dev.example.com", domains) {
		t.Fatalf("expected subdomain to require an explicit trusted domain")
	}

	if !EmailMatchesAllowedDomains("user@api.example.com", domains) {
		t.Fatalf("expected explicitly trusted subdomain to match")
	}
}
