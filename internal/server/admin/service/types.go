package service

import "errors"

var (
	ErrSmtpHostRequired                = errors.New("smtp host is required")
	ErrSmtpPortRequired                = errors.New("smtp port is required")
	ErrSmtpUsernameRequired            = errors.New("smtp username is required")
	ErrSmtpPasswordRequired            = errors.New("smtp password is required")
	ErrSmtpFromAddressRequired         = errors.New("smtp from address is required")
	ErrSmtpInviteEmailTemplateRequired = errors.New("smtp invite email template is required")
	ErrSmtpInviteEmailSubjectRequired  = errors.New("smtp invite email subject is required")
)

type UpdateSignupSettingsInput struct {
	SignupRequiresInvite           bool
	AllowRandomUserSignup          bool
	RandomUserSignupAllowedDomains string
}

type UpdateEmailSettingsInput struct {
	SmtpEnabled             bool
	SmtpHost                string
	SmtpPort                int32
	SmtpUsername            string
	SmtpPassword            string
	FromAddress             string
	UserInviteEmailTemplate string
	UserInviteEmailSubject  string
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
		if u.UserInviteEmailTemplate == "" {
			return ErrSmtpInviteEmailTemplateRequired
		}
		if u.UserInviteEmailSubject == "" {
			return ErrSmtpInviteEmailSubjectRequired
		}
	}
	return nil
}

type CreateInviteInput struct {
	Email string
	Role  string
}
