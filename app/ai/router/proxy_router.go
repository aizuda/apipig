package router

import (
	"apipig/app/ai/api"
	"github.com/gofiber/fiber/v2"
)

// ProxyRouter 负责注册代理池表相关路由。
type ProxyRouter struct{}

// InitProxyRouter 注册代理节点的增删改查和分页路由。
func (r *ProxyRouter) InitProxyRouter(router fiber.Router) {
	rg := router.Group("gateway/proxy")
	a := api.AiApi.ProxyApi
	rg.Post("save", a.SaveProxy)
	rg.Post("change-status", a.ChangeProxyStatus)
	rg.Post("delete", a.DeleteProxy)
	rg.Get("get", a.GetProxy)
	rg.Post("page", a.PageProxy)
}
