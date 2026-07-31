package router

import (
	"apipig/app/ai/api"
	"github.com/gofiber/fiber/v2"
)

// ProviderRouter 负责注册供应商表相关路由。
type ProviderRouter struct{}

// InitProviderRouter 注册供应商的增删改查和分页路由。
func (r *ProviderRouter) InitProviderRouter(router fiber.Router) {
	rg := router.Group("gateway/provider")
	a := api.AiApi.ProviderApi
	rg.Post("save", a.SaveProvider)
	rg.Post("change-status", a.ChangeProviderStatus)
	rg.Post("delete", a.DeleteProvider)
	rg.Get("get", a.GetProvider)
	rg.Post("list", a.ListProvider)
	rg.Post("page", a.PageProvider)
}
