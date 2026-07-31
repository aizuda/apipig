package router

import (
	"apipig/app/sys/api"
	"github.com/gofiber/fiber/v2"
)

type ResourceApiRouter struct {
}

func (r *ResourceApiRouter) InitResourceApiRouter(router fiber.Router) {
	rg := router.Group("resource-api")
	var a = api.SysApi.ResourceApiApi
	{
		rg.Post("create", a.Create)
		rg.Post("delete", a.Delete)
		rg.Post("update", a.Update)
		rg.Post("list", a.List)
	}
}
