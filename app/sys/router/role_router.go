package router

import (
	"apipig/app/sys/api"
	"github.com/gofiber/fiber/v2"
)

type RoleRouter struct {
}

func (r *RoleRouter) InitRoleRouter(router fiber.Router) {
	rg := router.Group("role")
	var a = api.SysApi.RoleApi
	{
		rg.Post("resource-set", a.ResourceSet)
		rg.Post("delete", a.Delete)
		rg.Post("update", a.Update)
		rg.Get("get", a.Get)
		rg.Post("list", a.List)
		rg.Post("page", a.Page)
	}
}
