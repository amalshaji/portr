package utils

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func GenerateTunnelSubdomain() string {
	id, _ := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz", 6)
	return id
}

func GenerateOAuthState() string {
	id, _ := gonanoid.New(32)
	return id
}

func GenerateSessionToken() string {
	id, _ := gonanoid.New(32)
	return id
}

func GenerateSecretKeyForUser() string {
	id, _ := gonanoid.New(42)
	return id
}
