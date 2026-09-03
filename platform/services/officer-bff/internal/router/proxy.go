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
	"strconv"
	"strings"
	"time"

	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
)

// signInternalContext podpisuje kontekst wewnętrzny HMAC dla komunikacji między mikroserwisami
//#region signInternalContext
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

//#region NewSingleHostProxy
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
				// Jeśli ścieżka zawiera parametr dynamiczny (np. UUID), podmieniamy tylko prefix
				if strings.Contains(targetPath, "{agreement_id}") {
					inPath := pr.In.URL.Path
					// Zamieniamy /api/v1/official/agreements/ na /api/v1/agreements/
					outPath := strings.Replace(inPath, "/api/v1/official/agreements/", "/api/v1/agreements/", 1)
					pr.Out.URL.Path = outPath
					pr.Out.URL.RawPath = outPath
				} else {
					// Dla sztywnych tras (np. /api/v1/citizens) wpisujemy dokładną ścieżkę
					pr.Out.URL.Path = targetPath
					pr.Out.URL.RawPath = targetPath
				}
			}

			if pr.In.Body != nil {
				bodyBytes, err := io.ReadAll(pr.In.Body)
				if err == nil && len(bodyBytes) > 0 {
					pr.In.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					pr.Out.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					pr.Out.ContentLength = int64(len(bodyBytes))
					pr.Out.Header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
				}
			}

			signInternalContext(pr.Out, keyStore, targetServiceID)
		},
	}

	return proxy.ServeHTTP, nil
}

// NewReverseProxy tworzy zwykłe proxy przelotowe (np. dla GET /auth/me oraz Krok 1 logowania)
//#region NewReverseProxy
func NewReverseProxy(authServiceURL, targetPath string, keyStore *httpserver.KeyStore) (http.HandlerFunc, error) {
	log := shared.GetLogger()

	target, err := url.Parse(authServiceURL)
	if err != nil {
		log.Error("Błąd parsowania authServiceURL w NewReverseProxy", "url", authServiceURL, "error", err)
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()

			// Jawne nadpisanie ścieżki docelowej w mikroserwisie
			pr.Out.URL.Path = targetPath
			pr.Out.URL.RawPath = targetPath

			signInternalContext(pr.Out, keyStore, "auth-service")
		},
	}

	return proxy.ServeHTTP, nil
}

//#region NewAuthTokenProxy
func NewAuthTokenProxy(authServiceURL, targetPath string, keyStore *httpserver.KeyStore, accessTokenTTL, refreshTokenTTL time.Duration) (http.HandlerFunc, error) {
	log := shared.GetLogger()

	target, err := url.Parse(authServiceURL)
	if err != nil {
		log.Error("Błąd parsowania authServiceURL w NewAuthTokenProxy", "url", authServiceURL, "error", err)
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()

			// Jawne nadpisanie ścieżki docelowej w mikroserwisie
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

			var responseData map[string]any
			if err := json.Unmarshal(body, &responseData); err != nil {
				log.Error("Błąd unmarshalingu odpowiedzi w proxy", "error", err)
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return nil
			}

			// Szukamy zagnieżdżonego obiektu "success" w odpowiedzi JSON
			if successMap, ok := responseData["success"].(map[string]any); ok {
				if accessToken, ok := successMap["access_token"].(string); ok && accessToken != "" {
					setAuthCookie(resp, "access_token", accessToken, accessTokenTTL, "/")
				}

				if refreshToken, ok := successMap["refresh_token"].(string); ok && refreshToken != "" {
					setAuthCookie(resp, "refresh_token", refreshToken, refreshTokenTTL, "/api/v1/official/auth/refresh")
				}

				// Usuwamy sekrety z obiektu przed wysłaniem do przeglądarki
				delete(successMap, "access_token")
				delete(successMap, "refresh_token")
			}

			cleanedBody, err := json.Marshal(responseData)
			if err != nil {
				log.Error("Błąd marshalingu oczyszczonej odpowiedzi w proxy", "error", err)
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return nil
			}

			resp.Body = io.NopCloser(bytes.NewReader(cleanedBody))
			resp.ContentLength = int64(len(cleanedBody))
			resp.Header.Set("Content-Length", strconv.Itoa(len(cleanedBody)))

			return nil
		},
	}

	return proxy.ServeHTTP, nil
}

//#region NewAuthLogoutProxy
func NewAuthLogoutProxy(authServiceURL, targetPath string, keyStore *httpserver.KeyStore) (http.HandlerFunc, error) {
	log := shared.GetLogger()

	target, err := url.Parse(authServiceURL)
	if err != nil {
		log.Error("Błąd parsowania authServiceURL w NewAuthLogoutProxy", "url", authServiceURL, "error", err)
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()

			// Jawne nadpisanie ścieżki docelowej w mikroserwisie
			pr.Out.URL.Path = targetPath
			pr.Out.URL.RawPath = targetPath

			signInternalContext(pr.Out, keyStore, "auth-service")
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode == http.StatusOK {
				clearAuthCookie(resp, "access_token")
				clearAuthCookie(resp, "refresh_token")
			}
			return nil
		},
	}

	return proxy.ServeHTTP, nil
}

// setAuthCookie ustawia bezpieczne ciasteczko HttpOnly
//#region setAuthCookie
func setAuthCookie(resp *http.Response, name, value string, duration time.Duration, path string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  time.Now().UTC().Add(duration),
		MaxAge:   int(duration.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	resp.Header.Add("Set-Cookie", cookie.String())
}

// clearAuthCookie unieważnia ciasteczko w przeglądarce
//#region clearAuthCookie
func clearAuthCookie(resp *http.Response, name string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	resp.Header.Add("Set-Cookie", cookie.String())
}
