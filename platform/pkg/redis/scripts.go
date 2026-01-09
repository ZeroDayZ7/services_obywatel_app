package redis

import (
	_ "embed"
)

//go:embed scripts/verify_2fa.lua
var verify2FAScript string
