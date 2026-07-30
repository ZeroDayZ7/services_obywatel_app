package router

import (
	"net/http"
	"strings"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true // CORS obsłużony na poziomie Gatewaya
	},
}

func handleWSProxy(container *di.Container, targetPath string) fiber.Handler {
	targetWS := container.Config.Services.WS
	// Normalizacja protokołu HTTP -> WS jeśli podano http:// w env
	targetWS = strings.Replace(targetWS, "http://", "ws://", 1)
	targetWS = strings.Replace(targetWS, "https://", "wss://", 1)

	log := shared.GetLogger()

	return func(c *fiber.Ctx) error {
		query := string(c.Request().URI().QueryString())
		ctx := c.Context()

		reqHeaders := http.Header{}

		if userID := c.Get(constants.HeaderUserID); userID != "" {
			reqHeaders.Set(constants.HeaderUserID, userID)
		}

		if token := c.Cookies(constants.CookieAccessToken); token != "" {
			reqHeaders.Set(constants.HeaderAuth, "Bearer "+token)
		}

		if queryToken := c.Query("token"); queryToken != "" {
			reqHeaders.Set(constants.HeaderAuth, "Bearer "+queryToken)
		}

		return upgrader.Upgrade(ctx, func(clientConn *websocket.Conn) {
			targetURL := targetWS + targetPath
			if len(query) > 0 {
				targetURL += "?" + query
			}

			log.Debug("[WS Proxy] Łączenie z wewnętrznym serwisem", "url", targetURL)

			dialer := websocket.Dialer{}
			backendConn, _, err := dialer.Dial(targetURL, reqHeaders)
			if err != nil {
				log.Error("[WS Proxy] Błąd połączenia z mikroserwisem docelowym", "error", err)
				_ = clientConn.Close()
				return
			}

			var once sync.Once
			closeConns := func() {
				_ = clientConn.Close()
				_ = backendConn.Close()
			}

			// Pętla Client -> Backend
			go func() {
				defer once.Do(closeConns)
				for {
					msgType, msg, err := clientConn.ReadMessage()
					if err != nil {
						break
					}
					if err := backendConn.WriteMessage(msgType, msg); err != nil {
						break
					}
				}
			}()

			// Pętla Backend -> Client
			go func() {
				defer once.Do(closeConns)
				for {
					msgType, msg, err := backendConn.ReadMessage()
					if err != nil {
						break
					}
					if err := clientConn.WriteMessage(msgType, msg); err != nil {
						break
					}
				}
			}()
		})
	}
}
