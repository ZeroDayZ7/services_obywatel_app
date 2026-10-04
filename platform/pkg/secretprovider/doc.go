// Package secretprovider abstracts credential retrieval away from the application.
//
// local:
//
//	development and test mode. Values are resolved from the local environment.
//
// kms12:
//
//	staging/production mode. The implementation delegates to the existing Secret Agent
//	bootstrap flow and keeps the application code dependent on the SecretProvider interface,
//	not on the agent implementation itself.
package secretprovider
