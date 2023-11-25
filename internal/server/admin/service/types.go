package service

type UpdateSignupSettingsInput struct {
	SignupRequiresInvite           bool
	AllowRandomUserSignup          bool
	RandomUserSignupAllowedDomains string
}

type UpdateEmailSettingsInput struct {
	UserInviteEmailTemplate string
}
