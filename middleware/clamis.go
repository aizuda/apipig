package middleware

import (
	"apipig/global"
	"github.com/gofiber/fiber/v2"
)

func GetTokenClaims(c *fiber.Ctx) *TokenClaims {
	if claims := c.Locals("tokenClaims"); claims == nil {
		global.LOG.Error("从Context中获取从jwt解析出来的用户信息失败, 请检查路由是否使用jwt中间件!")
		return nil
	} else {
		waitUse := claims.(*TokenClaims)
		return waitUse
	}
}
