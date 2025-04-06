package utils

import (
	"regexp"
	"strings"
)

var (
	slugifyRegex     = regexp.MustCompile("[^a-z0-9-]")
	multiHyphenRegex = regexp.MustCompile("-+")
)

func Trim(input string) string {
	return strings.TrimSpace(input)
}

func Slugify(s string) string {
	if s == "" {
		return s
	}

	s = strings.ToLower(s)

	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	s = slugifyRegex.ReplaceAllString(s, "")

	s = multiHyphenRegex.ReplaceAllString(s, "-")

	s = strings.Trim(s, "-")

	return s
}
