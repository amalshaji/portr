package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const randomFolderName = "portr_test_123456"

func TestEnsureDirExists(t *testing.T) {
	_ = EnsureDirExists(randomFolderName)
	defer os.Remove(randomFolderName)

	_, err := os.Stat(randomFolderName)
	assert.Nil(t, err)
}
