package api

import (
	"apipig/app/sys/model"
	sysReq "apipig/app/sys/model/request"
	"apipig/app/sys/service"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"github.com/gofiber/fiber/v2"
)

type RoleApi struct {
	api.API
}

var roleService = service.SysService.RoleService

// ResourceSet
// @Tags Role
// @Summary 角色创建修改并菜单权限分配
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.RoleResourceSetParams true "创建角色"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/role/resource-set [post]
func (a *RoleApi) ResourceSet(c *fiber.Ctx) error {
	params := &sysReq.RoleResourceSetParams{Ctx: c}
	err := a.BodyParserVerify(c, params, "创建角色")
	return response.Execute(c, roleService.ResourceSet, params, err)
}

// Delete
// @Tags Role
// @Summary 删除角色
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body request.IdsReq true "ID集合"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"删除成功"}"
// @Router /v1/sys/role/delete [post]
func (a *RoleApi) Delete(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除角色")
	return response.Execute(c, roleService.Delete, idsReq, err)
}

// Update
// @Tags Role
// @Summary 更新角色
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body model.Role true "角色信息"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"设置成功"}"
// @Router /v1/sys/role/update [post]
func (a *RoleApi) Update(c *fiber.Ctx) error {
	m := new(model.Role)
	err := a.BodyParserIdVerify(c, m, "角色信息")
	return response.Execute(c, roleService.Update, m, err)
}

// Get
// @Tags Role
// @Summary 根据id获取角色
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param id query string true "角色ID"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/role/get [get]
func (a *RoleApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, roleService.GetRoleRespById, id, err)
}

// List
// @Tags Role
// @Summary 获取角色列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.RoleParams true "角色列表查询参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/role/list [post]
func (a *RoleApi) List(c *fiber.Ctx) error {
	var params *sysReq.RoleParams
	err := a.BodyParser(c, &params, "角色列表")
	return response.Execute(c, roleService.List, params, err)
}

// Page
// @Tags Role
// @Summary 分页获取角色列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.RolePageParams true "页码, 每页大小等分页参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/role/page [post]
func (a *RoleApi) Page(c *fiber.Ctx) error {
	var pageInfo *sysReq.RolePageParams
	err := a.BodyParser(c, &pageInfo, "角色分页")
	return response.Execute(c, roleService.Page, pageInfo, err)
}
