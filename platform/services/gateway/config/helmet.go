package config

import "github.com/gofiber/fiber/v2/middleware/helmet"

func HelmetConfig() helmet.Config {
	return helmet.Config{
		XSSProtection:             "0",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "DENY",
		HSTSMaxAge:                31536000,
		HSTSExcludeSubdomains:     false,
		HSTSPreloadEnabled:        false,
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self'; object-src 'none';",
		CSPReportOnly:             false,
		ReferrerPolicy:            "no-referrer",
		PermissionPolicy:          "",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
	}
}
