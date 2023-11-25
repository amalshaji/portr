package service

type UpdateSettingsInput struct {
	SignupRequiresInvite           bool
	AllowRandomUserSignup          bool
	RandomUserSignupAllowedDomains string
}
