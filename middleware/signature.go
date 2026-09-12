package middleware

import "github.com/gofiber/fiber/v2"

func SignatureAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
