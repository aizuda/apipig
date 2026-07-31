package router

import (
	"apipig/app/ai/api"
	"github.com/gofiber/fiber/v2"
)

// CallLogRouter 负责注册调用日志表相关路由。
type CallLogRouter struct{}

// InitCallLogRouter 注册调用日志分页查询路由。
func (r *CallLogRouter) InitCallLogRouter(router fiber.Router) {
	rg := router.Group("gateway/log")
	rg.Post("page", api.AiApi.CallLogApi.PageCallLog)
}
