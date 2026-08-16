package router

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewSingleHostProxy(targetURL string) (http.HandlerFunc, error) {
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
	}

	return proxy.ServeHTTP, nil
}

func NewAuthLoginProxy(authServiceURL string) (http.HandlerFunc, error) {
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
