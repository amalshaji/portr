package service

import "errors"

var (
	ErrSmtpHostRequired                = errors.New("smtp host is required")
	ErrSmtpPortRequired                = errors.New("smtp port is required")
	ErrSmtpUsernameRequired            = errors.New("smtp username is required")
	ErrSmtpPasswordRequired            = errors.New("smtp password is required")
	ErrSmtpFromAddressRequired         = errors.New("smtp from address is required")
	ErrSmtpInviteEmailSubjectRequired  = errors.New("smtp invite email subject is required")
	ErrSmtpInviteEmailTemplateRequired = errors.New("smtp invite email template is required")
)

type UpdateSignupSettingsInput struct {
	SignupRequiresInvite           bool
	AllowRandomUserSignup          bool
	RandomUserSignupAllowedDomains string
}

type UpdateEmailSettingsInput struct {
	SmtpEnabled            bool
	SmtpHost               string
	SmtpPort               int32
	SmtpUsername           string
	SmtpPassword           string
	FromAddress            string
	AddMemberEmailTemplate string
	AddMemberEmailSubject  string
}

func (u UpdateEmailSettingsInput) Validate() error {
	if u.SmtpEnabled {
		if u.SmtpHost == "" {
			return ErrSmtpHostRequired
		}
		if u.SmtpPort == 0 {
			return ErrSmtpPortRequired
		}
		if u.SmtpUsername == "" {
			return ErrSmtpUsernameRequired
		}
		if u.SmtpPassword == "" {
			return ErrSmtpPasswordRequired
		}
		if u.FromAddress == "" {
			return ErrSmtpFromAddressRequired
		}
		if u.AddMemberEmailSubject == "" {
			return ErrSmtpInviteEmailSubjectRequired
		}
		if u.AddMemberEmailTemplate == "" {
			return ErrSmtpInviteEmailTemplateRequired
		}
	}
	return nil
}

type AddMemberInput struct {
	Email string
	Role  string
}
