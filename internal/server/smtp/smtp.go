package smtp

import (
	"log/slog"
	"net/smtp"

	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/utils"
)

type Smtp struct {
	config *config.AdminConfig
	log    *slog.Logger
}

func New(config *config.AdminConfig) *Smtp {
	return &Smtp{config: config, log: utils.GetLogger()}
}

type SendEmailInput struct {
	From    string
	To      string
	Subject string
	Body    string
}

func (s *Smtp) SendEmail(input SendEmailInput) error {
	auth := smtp.PlainAuth("", s.config.Smtp.Username, s.config.Smtp.Password, s.config.Smtp.Host)
	return smtp.SendMail(s.config.Smtp.Address(), auth, input.From, []string{input.To}, []byte(input.Body))
}
