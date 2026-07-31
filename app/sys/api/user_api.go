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

type UserApi struct {
	api.API
}

var userService = service.SysService.UserService

// AssignSet
// @Tags User
// @Summary 用户分配设置
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.UserAssignSetParams true "用户分配设置参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/user/assign-set [post]
func (a *UserApi) AssignSet(c *fiber.Ctx) error {
	params := &sysReq.UserAssignSetParams{Ctx: c}
	err := a.BodyParserVerify(c, params, "创建用户")
	return response.Execute(c, userService.AssignSet, params, err)
}

// Delete
// @Tags User
// @Summary 删除用户
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body request.IdsReq true "ID集合"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"删除成功"}"
// @Router /v1/sys/user/delete [post]
func (a *UserApi) Delete(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除用户")
	return response.Execute(c, userService.Delete, idsReq, err)
}

// Update
// @Tags User
// @Summary 更新用户
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body model.User true "用户信息"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"设置成功"}"
// @Router /v1/sys/user/update [post]
func (a *UserApi) Update(c *fiber.Ctx) error {
	m := new(model.User)
	err := a.BodyParserIdVerify(c, m, "用户信息")
	return response.Execute(c, userService.Update, m, err)
}

// ResetPassword
// @Tags User
// @Summary 密码重置
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.ResetPasswordParams true "密码重置"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"设置成功"}"
// @Router /v1/sys/user/reset-password [post]
func (a *UserApi) ResetPassword(c *fiber.Ctx) error {
	params := new(sysReq.ResetPasswordParams)
	err := a.BodyParserVerify(c, params, "密码重置")
	return response.Execute(c, userService.ResetPassword, params, err)
}

// AssignRoles
// @Tags User
// @Summary 分配角色
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.AssignRolesParams true "分配角色"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"设置成功"}"
// @Router /v1/sys/user/assign-roles [post]
func (a *UserApi) AssignRoles(c *fiber.Ctx) error {
	params := new(sysReq.AssignRolesParams)
	err := a.BodyParserVerify(c, params, "分配角色")
	return response.Execute(c, userService.AssignRoles, params, err)
}

// Get
// @Tags User
// @Summary 根据id获取用户
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param id query string true "用户ID"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/user/get [get]
func (a *UserApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, userService.GetUserRespById, id, err)
}

// Info
// @Tags User
// @Summary 获取当前用户信息
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/user/info [get]
func (a *UserApi) Info(c *fiber.Ctx) error {
	return response.Execute(c, userService.GetUserInfo, c, nil)
}

// List
// @Tags User
// @Summary 获取用户列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.UserParams true "用户列表查询参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/user/list [post]
func (a *UserApi) List(c *fiber.Ctx) error {
	var params *sysReq.UserParams
	err := a.BodyParser(c, &params, "用户列表")
	return response.Execute(c, userService.List, params, err)
}

// Page
// @Tags User
// @Summary 分页获取用户列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.UserPageParams true "页码, 每页大小等分页参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/user/page [post]
func (a *UserApi) Page(c *fiber.Ctx) error {
	var pageInfo *sysReq.UserPageParams
	err := a.BodyParser(c, &pageInfo, "用户分页")
	return response.Execute(c, userService.Page, pageInfo, err)
}
