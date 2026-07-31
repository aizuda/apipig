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

type ResourceApiApi struct {
	api.API
}

var resourceApiService = service.SysService.ResourceApiService

// Create
// @Tags ResourceApi
// @Summary 创建资源权限API
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.ResourceApiCreateParams true "创建资源权限API"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/resource-api/create [post]
func (a *ResourceApiApi) Create(c *fiber.Ctx) error {
	params := &sysReq.ResourceApiCreateParams{Ctx: c}
	err := a.BodyParserVerify(c, params, "创建资源权限API")
	return response.Execute(c, resourceApiService.Create, params, err)
}

// Delete
// @Tags ResourceApi
// @Summary 删除资源权限API
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body request.IdsReq true "权限API ID集合"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"删除成功"}"
// @Router /v1/sys/resource-api/delete [post]
func (a *ResourceApiApi) Delete(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除资源权限API")
	return response.Execute(c, resourceApiService.Delete, idsReq, err)
}

// Update
// @Tags ResourceApi
// @Summary 更新资源权限API
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body model.ResourceApi true "资源权限API信息"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"设置成功"}"
// @Router /v1/sys/resource-api/update [post]
func (a *ResourceApiApi) Update(c *fiber.Ctx) error {
	m := new(model.ResourceApi)
	err := a.BodyParserIdVerify(c, m, "更新资源权限API信息")
	return response.Execute(c, resourceApiService.Update, m, err)
}

// List
// @Tags ResourceApi
// @Summary 资源权限API列表
// @Security ApiKeyAuth
// @accept json
// @Produce json
// @Param data body sysReq.ResourceApiParams true "资源权限API列表查询参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"获取成功"}"
// @Router /v1/sys/resource-api/list [post]
func (a *ResourceApiApi) List(c *fiber.Ctx) error {
	var params *sysReq.ResourceApiParams
	err := a.BodyParser(c, &params, "资源权限API列表")
	return response.Execute(c, resourceApiService.ListByResourceId, params, err)
}
