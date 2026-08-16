package router

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/httpserver"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func signInternalContext(req *http.Request, keyStore *httpserver.KeyStore, targetServiceID string) {
	internalCtx := req.Header.Get(constants.HeaderInternalContext)
	if internalCtx == "" {
		return
	}

	secret, _, ok := keyStore.GetKey(targetServiceID)
	if !ok {
		return
	}

	payload, err := base64.StdEncoding.DecodeString(internalCtx)
	if err != nil {
		return
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set(constants.HeaderInternalSignature, signature)
	req.Header.Set("X-Internal-Service", "officer-bff")
}

func NewSingleHostProxy(targetURL, targetPath, targetServiceID string, keyStore *httpserver.KeyStore) (http.HandlerFunc, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = target.Host

		if targetPath != "" {
			req.URL.Path = targetPath
			req.URL.RawPath = targetPath
		}

		signInternalContext(req, keyStore, targetServiceID)
	}

	return proxy.ServeHTTP, nil
}

func NewAuthLoginProxy(authServiceURL, targetPath string, keyStore *httpserver.KeyStore) (http.HandlerFunc, error) {
	target, err := url.Parse(authServiceURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = target.Host

		req.URL.Path = targetPath
		req.URL.RawPath = targetPath

		signInternalContext(req, keyStore, "auth-service")
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()

		var tokens tokenResponse
		if err := json.Unmarshal(body, &tokens); err == nil && tokens.AccessToken != "" {
			setAuthCookie(resp, "access_token", tokens.AccessToken, 15*time.Minute)
			if tokens.RefreshToken != "" {
				setAuthCookie(resp, "refresh_token", tokens.RefreshToken, 7*24*time.Hour)
			}
		}

		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
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
