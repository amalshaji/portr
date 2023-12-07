package smtp

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
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
	auth := sasl.NewPlainClient("", s.config.Smtp.Username, s.config.Smtp.Password)
	message := fmt.Sprintf("Subject: %s\n\n%s", input.Subject, input.Body)
	return smtp.SendMail(
		s.config.Smtp.Address(),
		auth, input.From,
		[]string{input.To},
		strings.NewReader(message),
	)
}
