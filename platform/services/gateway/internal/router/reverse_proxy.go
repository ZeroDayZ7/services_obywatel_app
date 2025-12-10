package router

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ReverseProxy przesyła request do wskazanego adresu backendowego
func ReverseProxy(target string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()

		req, err := http.NewRequest(
			string(c.Method()),
			target+c.OriginalURL(),
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}

		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			for _, vv := range v {
				c.Set(k, vv)
			}
		}

		c.Status(resp.StatusCode)
		_, err = io.Copy(c, resp.Body)
		return err
	}
}

// ReverseProxyWithUserID - przekazuje request do backendu i dodaje X-User-ID z ctx
func ReverseProxySecure(target string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()

		req, err := http.NewRequest(
			string(c.Method()),
			target+c.OriginalURL(),
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}

		// Kopiujemy TYLKO krytyczne nagłówki
		if ct := c.Get("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		req.Header.Set("Accept", c.Get("Accept", "*/*"))

		// Dodajemy dane z gateway
		if uid := c.Locals("userID"); uid != nil {
			req.Header.Set("X-User-Id", uid.(string))
		}
		if sid := c.Locals("sessionID"); sid != nil {
			req.Header.Set("X-Session-Id", sid.(string))
		}

		// Wysłanie requesta
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Kopiujemy tylko sensowne response nagłówki
		for _, h := range []string{"Content-Type"} {
			if v := resp.Header.Get(h); v != "" {
				c.Set(h, v)
			}
		}

		c.Status(resp.StatusCode)
		_, err = io.Copy(c, resp.Body)
		return err
	}
}

// ReverseProxyWithCustomHeader - pozwala przekazać dowolny nagłówek z ctx do backendu
func ReverseProxyWithCustomHeader(target string, headerKey string, ctxKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()

		req, err := http.NewRequest(
			string(c.Method()),
			target+c.OriginalURL(),
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}

		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// Dodanie nagłówka z ctx
		if val := c.Locals(ctxKey); val != nil {
			req.Header.Set(headerKey, val.(string))
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			for _, vv := range v {
				c.Set(k, vv)
			}
		}

		c.Status(resp.StatusCode)
		_, err = io.Copy(c, resp.Body)
		return err
	}
}
