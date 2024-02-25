package utils

import (
	"regexp"
	"strings"
)

func Trim(input string) string {
	return strings.TrimSpace(input)
}

func Slugify(s string) string {
	// Convert the string to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove special characters using regular expression
	reg := regexp.MustCompile("[^a-z0-9-]")
	s = reg.ReplaceAllString(s, "")

	// Remove consecutive hyphens
	s = strings.ReplaceAll(s, "--", "-")

	// Remove leading and trailing hyphens
	s = strings.Trim(s, "-")

	return s
}
