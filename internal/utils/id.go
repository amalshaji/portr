package utils

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func GenerateTunnelSubdomain() string {
	id, _ := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz", 6)
	return id
}
