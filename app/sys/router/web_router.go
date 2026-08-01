package router

import (
	"apipig/app/sys/api"
	"github.com/gofiber/fiber/v2"
)

type WebRouter struct {
}

func (r *WebRouter) InitWebRouter(router fiber.Router) {
	var a = api.SysApi.WebApi
	{
		router.Get("captcha", a.Captcha)
		router.Post("public-key", a.PublicKey)
		router.Post("login", a.Login)
		router.Post("token-authorize", a.AuthorizeToken)
		router.Post("refresh-token", a.RefreshToken)

		// 测试接口
		router.Post("test-http-push", a.TestHttpPush)
	}
}
