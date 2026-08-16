package router

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func signInternalContext(req *http.Request, keyStore *httpserver.KeyStore, targetServiceID string) {
	log := shared.GetLogger()

	internalCtx := req.Header.Get(constants.HeaderInternalContext)
	if internalCtx == "" {
		log.Warn("Brak nagłówka X-Internal-Context w żądaniu wychodzącym")
		return
	}

	secret, _, ok := keyStore.GetKey(targetServiceID)
	if !ok {
		log.Error("Nie znaleziono klucza HMAC w KeyStore", "targetServiceID", targetServiceID)
		return
	}

	payload, err := base64.StdEncoding.DecodeString(internalCtx)
	if err != nil {
		log.Error("Błąd dekodowania Base64 dla X-Internal-Context", "error", err)
		return
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req.Header.Set(constants.HeaderInternalSignature, signature)
	req.Header.Set("X-Internal-Service", "officer-bff")
}

func NewSingleHostProxy(targetURL, targetPath, targetServiceID string, keyStore *httpserver.KeyStore) (http.HandlerFunc, error) {
	log := shared.GetLogger()

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Error("Błąd parsowania targetURL w NewSingleHostProxy", "url", targetURL, "error", err)
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()

			if targetPath != "" {
				pr.Out.URL.Path = targetPath
				pr.Out.URL.RawPath = targetPath
			}

			signInternalContext(pr.Out, keyStore, targetServiceID)
		},
	}

	return proxy.ServeHTTP, nil
}

func NewAuthLoginProxy(authServiceURL, targetPath string, keyStore *httpserver.KeyStore) (http.HandlerFunc, error) {
	log := shared.GetLogger()

	target, err := url.Parse(authServiceURL)
	if err != nil {
		log.Error("Błąd parsowania authServiceURL w NewAuthLoginProxy", "url", authServiceURL, "error", err)
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()

			pr.Out.URL.Path = targetPath
			pr.Out.URL.RawPath = targetPath

			signInternalContext(pr.Out, keyStore, "auth-service")
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode != http.StatusOK {
				return nil
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error("Błąd odczytu odpowiedzi z auth-service", "error", err)
				return err
			}
			_ = resp.Body.Close()

			var tokens tokenResponse
			if err := json.Unmarshal(body, &tokens); err == nil && tokens.AccessToken != "" {
				setAuthCookie(resp, "access_token", tokens.AccessToken, 15*time.Minute)
				if tokens.RefreshToken != "" {
					setAuthCookie(resp, "refresh_token", tokens.RefreshToken, 7*24*time.Hour)
				}
			} else if err != nil {
				log.Error("Błąd unmarshalingu tokenów w proxy", "error", err)
			}

			resp.Body = io.NopCloser(bytes.NewReader(body))
			return nil
		},
	}

	return proxy.ServeHTTP, nil
}

func setAuthCookie(resp *http.Response, name, value string, duration time.Duration) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(duration),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	resp.Header.Add("Set-Cookie", cookie.String())
}
