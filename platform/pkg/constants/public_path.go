package constants

var PublicPaths = []string{
	"/auth/login",
	"/auth/register",
	"/auth/refresh",
	"/auth/2fa-verify",
	"/auth/2fa-resend",
	"/auth/reset/send",
	"/auth/reset/verify",
	"/auth/reset/final",
	"/health",
	"/api/v1/official/auth/login",
}

var DeviceVerifyPaths = []string{
	"/auth/register-device",
	"/auth/verify-device",
	"/auth/temporary-session",
	"/api/v1/official/auth/login/step2",
}
