package models

import "testing"

func TestNormalizeAutoSignupDomain(t *testing.T) {
	got, ok := NormalizeAutoSignupDomain(" @Example.com. ")
	if !ok {
		t.Fatalf("expected domain to normalize")
	}

	want := "example.com"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeAutoSignupDomainRejectsInvalidDomains(t *testing.T) {
	invalidDomains := []string{
		"",
		"user@example.com",
		"example.com,acme.co",
		"dev..example.com",
	}

	for _, domain := range invalidDomains {
		if got, ok := NormalizeAutoSignupDomain(domain); ok {
			t.Fatalf("expected %q to be invalid, got %q", domain, got)
		}
	}
}

func TestEmailDomainRequiresExactDomain(t *testing.T) {
	domain, ok := EmailDomain("user@api.example.com.")
	if !ok {
		t.Fatalf("expected email domain to parse")
	}

	if domain != "api.example.com" {
		t.Fatalf("expected explicit email domain, got %q", domain)
	}
}
