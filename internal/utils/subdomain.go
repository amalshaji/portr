package utils

import (
	"fmt"
	"regexp"
)

func ValidateSubdomain(subdomain string) error {
	matched, err := regexp.Match(`^[a-zA-Z0-9][-a-zA-Z0-9_]{0,61}[a-zA-Z0-9]$`, []byte(subdomain))
	if err != nil {
		return fmt.Errorf("error validating subdomain: %v", err)
	}
	if !matched {
		return fmt.Errorf("invalid subdomain '%s'. Must not contain special characters other than '-', `_`", subdomain)
	}

	return nil
}
