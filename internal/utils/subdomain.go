package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var subdomainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func NormalizeSubdomain(subdomain string) string {
	return strings.ToLower(strings.TrimSpace(subdomain))
}

func ValidateSubdomain(subdomain string) error {
	if !subdomainPattern.MatchString(subdomain) {
		return fmt.Errorf("invalid subdomain %q: use 1-63 lowercase letters, numbers, or internal hyphens", subdomain)
	}

	return nil
}
