package router

import (
	"apipig/app/ai/api"
	"github.com/gofiber/fiber/v2"
)

// AccessTokenRouter 负责注册访问令牌表相关路由。
type AccessTokenRouter struct{}

// InitAccessTokenRouter 注册访问令牌的增删改查和分页路由。
func (r *AccessTokenRouter) InitAccessTokenRouter(router fiber.Router) {
	rg := router.Group("gateway/token")
	a := api.AiApi.AccessTokenApi
	rg.Post("save", a.SaveAccessToken)
	rg.Post("update-tags", a.UpdateAccessTokenTags)
	rg.Post("change-status", a.ChangeAccessTokenStatus)
	rg.Post("delete", a.DeleteAccessToken)
	rg.Get("get", a.GetAccessToken)
	rg.Post("page", a.PageAccessToken)
}
