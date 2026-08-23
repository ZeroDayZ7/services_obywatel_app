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

// #region signInternalContext
// signInternalContext podpisuje kontekst wewnętrzny HMAC dla komunikacji między mikroserwisami
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

// #region NewSingleHostProxy
// NewSingleHostProxy tworzy proxy, które zachowuje dynamiczną końcówkę ścieżki (np. ID i akcję)
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

			// 1. Pobieramy oryginalną ścieżkę z żądania przychodzącego do BFF
			originalPath := pr.In.URL.Path // np. /api/v1/official/agreements/01a02c7f-.../download

			// 2. Usuwamy prefiks BFF (/api/v1/official)
			trimmed := strings.TrimPrefix(originalPath, "/api/v1/official") // np. /agreements/01a02c7f-.../download

			// 3. Określamy, jaki zasób obsługujemy (np. /agreements lub /citizens) na podstawie trimmed
			var suffix string
			if strings.HasPrefix(trimmed, "/agreements") {
				suffix = strings.TrimPrefix(trimmed, "/agreements")
			} else if strings.HasPrefix(trimmed, "/citizens") {
				suffix = strings.TrimPrefix(trimmed, "/citizens")
			} else {
				suffix = trimmed
			}

			// 4. Składamy docelową ścieżkę dla mikroserwisu (np. /api/v1/agreements + /{id}/download)
			pr.Out.URL.Path = targetPath + suffix
			pr.Out.URL.RawPath = pr.Out.URL.Path

			signInternalContext(pr.Out, keyStore, targetServiceID)
		},
	}

	return proxy.ServeHTTP, nil
}

// #endregion

// #region NewReverseProxy
// NewReverseProxy tworzy zwykłe proxy przelotowe (np. dla GET /auth/me oraz Krok 1 logowania)
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

// #endregion

// #region NewAuthTokenProxy
// NewAuthTokenProxy przechwytuje odpowiedź z auth-service, wyciąga access_token/refresh_token,
// zapisuje je w bezpiecznych ciasteczkach HttpOnly oraz wycina je z odpowiedzi JSON dla Angulara.
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

			if accessToken, ok := responseData["access_token"].(string); ok && accessToken != "" {
				setAuthCookie(resp, "access_token", accessToken, accessTokenTTL, "/")
			}

			if refreshToken, ok := responseData["refresh_token"].(string); ok && refreshToken != "" {
				setAuthCookie(resp, "refresh_token", refreshToken, refreshTokenTTL, "/api/v1/official/auth/refresh")
			}

			delete(responseData, "access_token")
			delete(responseData, "refresh_token")

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

// #endregion

// #region NewAuthLogoutProxy
// NewAuthLogoutProxy przekazuje żądanie wylogowania i czyści ciasteczka w przeglądarce
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

// #endregion

// #region Helpers (Cookies)
// setAuthCookie ustawia bezpieczne ciasteczko HttpOnly
func setAuthCookie(resp *http.Response, name, value string, duration time.Duration, path string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  time.Now().Add(duration),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	resp.Header.Add("Set-Cookie", cookie.String())
}

// clearAuthCookie unieważnia ciasteczko w przeglądarce
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

// #endregion
