package router

import (
	"apipig/app/ai/api"

	"github.com/gofiber/fiber/v2"
)

// ChannelAccountRouter 负责注册渠道账户管理路由。
type ChannelAccountRouter struct{}

func (r *ChannelAccountRouter) InitChannelAccountRouter(router fiber.Router) {
	rg := router.Group("gateway/channel-account")
	a := api.AiApi.ChannelAccountApi
	rg.Post("save", a.SaveChannelAccount)
	rg.Post("change-status", a.ChangeChannelAccountStatus)
	rg.Post("delete", a.DeleteChannelAccount)
	rg.Get("get", a.GetChannelAccount)
	rg.Post("page", a.PageChannelAccount)
}
