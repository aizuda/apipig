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

type ResourceApi struct {
	api.API
}

var resourceService = service.SysService.ResourceService

// Create
// @Tags Resource
// @Summary 创建资源
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.ResourceCreateParams true "创建资源"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/resource/create [post]
func (a *ResourceApi) Create(c *fiber.Ctx) error {
	params := &sysReq.ResourceCreateParams{Ctx: c}
	err := a.BodyParserVerify(c, params, "创建资源")
	return response.Execute(c, resourceService.Create, params, err)
}

// Delete
// @Tags Resource
// @Summary 删除资源
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body request.IdsReq true "ID集合"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"删除成功"}"
// @Router /v1/sys/resource/delete [post]
func (a *ResourceApi) Delete(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除资源")
	return response.Execute(c, resourceService.Delete, idsReq, err)
}

// Update
// @Tags Resource
// @Summary 更新资源
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body model.Resource true "资源信息"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"设置成功"}"
// @Router /v1/sys/resource/update [post]
func (a *ResourceApi) Update(c *fiber.Ctx) error {
	m := new(model.Resource)
	err := a.BodyParserIdVerify(c, m, "资源信息")
	return response.Execute(c, resourceService.Update, m, err)
}

// Get
// @Tags Resource
// @Summary 根据id获取资源
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param id query string true "资源ID"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/resource/get [get]
func (a *ResourceApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, resourceService.GetById, id, err)
}

// ListMenu
// @Tags Resource
// @Summary 获取菜单列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/resource/list-menu [post]
func (a *ResourceApi) ListMenu(c *fiber.Ctx) error {
	return response.Execute(c, resourceService.ListMenu, c, nil)
}

// ListTree
// @Tags Resource
// @Summary 获取资源树列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.ResourceParams true "资源树列表查询参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/resource/list-tree [post]
func (a *ResourceApi) ListTree(c *fiber.Ctx) error {
	var params *sysReq.ResourceParams
	err := a.BodyParser(c, &params, "资源树列表")
	return response.Execute(c, resourceService.ListTree, params, err)
}
