package service

type UpdateSignupSettingsInput struct {
	SignupRequiresInvite           bool
	AllowRandomUserSignup          bool
	RandomUserSignupAllowedDomains string
}

type UpdateEmailSettingsInput struct {
	UserInviteEmailTemplate string
}

type CreateInviteInput struct {
	Email string
	Role  string
}
