package services

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandomPassword() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
