package router

import (
	"apipig/app/sys/api"

	"github.com/gofiber/fiber/v2"
)

type UserRouter struct {
}

func (r *UserRouter) InitUserRouter(router fiber.Router) {
	rg := router.Group("user")
	var a = api.SysApi.UserApi
	{
		rg.Post("assign-set", a.AssignSet)
		rg.Post("delete", a.Delete)
		rg.Post("update", a.Update)
		rg.Post("reset-password", a.ResetPassword)
		rg.Post("assign-roles", a.AssignRoles)
		rg.Get("get", a.Get)
		rg.Get("info", a.Info)
		rg.Post("list", a.List)
		rg.Post("page", a.Page)
	}
}
