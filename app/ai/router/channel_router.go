package router

import (
	"apipig/app/ai/api"
	"github.com/gofiber/fiber/v2"
)

// ChannelRouter 负责注册渠道账号表相关路由。
type ChannelRouter struct{}

// InitChannelRouter 注册渠道账号的增删改查和分页路由。
func (r *ChannelRouter) InitChannelRouter(router fiber.Router) {
	rg := router.Group("gateway/channel")
	a := api.AiApi.ChannelApi
	rg.Post("save", a.SaveChannel)
	rg.Post("change-status", a.ChangeChannelStatus)
	rg.Post("delete", a.DeleteChannel)
	rg.Get("get", a.GetChannel)
	rg.Post("page", a.PageChannel)
}
