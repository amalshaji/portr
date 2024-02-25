package utils

import (
	"log/slog"
	"os"
)

func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
