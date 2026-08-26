package kms

import (
	"net/http"
	"time"
)

const (
	DefaultTimeout = 5 * time.Second

	EndpointHealth          = "/health"
	EndpointGenerate        = "/api/v1/keys/generate"
	EndpointGenerateDataKey = "/api/v1/keys/generate-data-key"
	EndpointPublic          = "/api/v1/keys/public/%s/%s"
	EndpointSymmetric       = "/api/v1/keys/symmetric"

	HeaderServiceName   = "X-Service-Name"
	HeaderTimestamp     = "X-Timestamp"
	HeaderNonce         = "X-Nonce"
	HeaderBodySHA256    = "X-Body-SHA256"
	HeaderHMACSignature = "X-HMAC-Signature"
	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"

	MIMEApplicationJSON = "application/json"
	DefaultAlgorithm    = "Ed25519"
	AlgorithmAES256GCM  = "AES256GCM"
)

type Config struct {
	Endpoint      string        `json:"endpoint"`
	ServiceName   string        `json:"service_name"`
	ServiceSecret string        `json:"service_secret"`
	Timeout       time.Duration `json:"timeout"`
	HTTPClient    *http.Client  `json:"-"`
}
