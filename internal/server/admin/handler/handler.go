package handler

import "github.com/amalshaji/localport/internal/server/config"

type Handler struct {
	config *config.AdminConfig
}

func New(config *config.AdminConfig) *Handler {
	return &Handler{config: config}
}
