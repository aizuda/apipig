package router

import (
	"apipig/app/ai/api"

	"github.com/gofiber/fiber/v2"
)

// AccessTokenTagRouter 负责注册 API 密钥标签管理路由。
type AccessTokenTagRouter struct{}

// InitAccessTokenTagRouter 注册 API 密钥标签增删改查和排序路由。
func (r *AccessTokenTagRouter) InitAccessTokenTagRouter(router fiber.Router) {
	rg := router.Group("gateway/token-tag")
	a := api.AiApi.AccessTokenTagApi
	rg.Post("save", a.SaveAccessTokenTag)
	rg.Post("delete", a.DeleteAccessTokenTag)
	rg.Post("list", a.ListAccessTokenTag)
	rg.Post("sort", a.SortAccessTokenTag)
}
