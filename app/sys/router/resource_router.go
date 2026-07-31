package router

import (
	"apipig/app/sys/api"
	"github.com/gofiber/fiber/v2"
)

type ResourceRouter struct {
}

func (r *ResourceRouter) InitResourceRouter(router fiber.Router) {
	rg := router.Group("resource")
	var a = api.SysApi.ResourceApi
	{
		rg.Post("create", a.Create)
		rg.Post("delete", a.Delete)
		rg.Post("update", a.Update)
		rg.Get("get", a.Get)
		rg.Post("list-menu", a.ListMenu)
		rg.Post("list-tree", a.ListTree)
	}
}
