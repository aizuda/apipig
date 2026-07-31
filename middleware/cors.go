package middleware

import (
	"apipig/global"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// Cors 处理跨域请求,支持options访问
func Cors() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: global.CONFIG.System.AllowOrigins,
	})
}
