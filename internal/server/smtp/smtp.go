package smtp

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/amalshaji/localport/internal/server/config"
	db "github.com/amalshaji/localport/internal/server/db/models"
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

func (s *Smtp) SendEmail(input SendEmailInput, settings *db.GlobalSetting) error {
	auth := sasl.NewPlainClient("", settings.SmtpUsername.(string), settings.SmtpPassword.(string))
	message := fmt.Sprintf("Subject: %s\n\n%s", input.Subject, input.Body)
	return smtp.SendMail(
		settings.SmtpHost.(string)+":"+fmt.Sprint(settings.SmtpPort.(int64)),
		auth, input.From,
		[]string{input.To},
		strings.NewReader(message),
	)
}
