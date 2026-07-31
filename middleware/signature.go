package middleware

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

func SignatureAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 签名值
		fmt.Println(c.Params("signature"))
		// 时间戳
		fmt.Println(c.Params("timestamp"))
		// 随机数
		fmt.Println(c.Params("nonce"))
		return c.Next()
	}
}
