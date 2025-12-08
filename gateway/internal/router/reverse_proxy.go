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
		// Pobranie body requestu
		body := c.Body() // <- tu cała zawartość body

		// Tworzymy nowy request HTTP
		req, err := http.NewRequest(
			string(c.Method()),
			target+c.OriginalURL(),
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}

		// Skopiowanie headerów z oryginalnego requestu
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// Wykonanie requestu
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Skopiowanie odpowiedzi z backendu do klienta
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
