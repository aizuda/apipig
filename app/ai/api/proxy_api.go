package api

import (
	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/app/ai/service"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

// ProxyApi 负责 AI 代理池表的管理接口。
type ProxyApi struct {
	api.API
	service *service.ProxyService
}

func NewProxyApi(proxyService *service.ProxyService) *ProxyApi {
	return &ProxyApi{service: proxyService}
}

// SaveProxy 创建或更新代理节点。
func (a *ProxyApi) SaveProxy(c *fiber.Ctx) error {
	m := new(model.Proxy)
	err := a.BodyParserVerify(c, m, "代理节点")
	return response.Execute(c, a.service.Save, &aiReq.ProxySaveParams{Ctx: c, Proxy: m}, err)
}

// ChangeProxyStatus 切换代理节点启用、禁用状态。
func (a *ProxyApi) ChangeProxyStatus(c *fiber.Ctx) error {
	params := new(aiReq.StatusChangeParams)
	err := a.BodyParserVerify(c, params, "代理状态")
	return response.Execute(c, a.service.ChangeStatus, params, err)
}

// DeleteProxy 批量删除代理节点。
func (a *ProxyApi) DeleteProxy(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除代理")
	return response.Execute(c, a.service.Delete, idsReq, err)
}

// GetProxy 根据 ID 查询代理节点。
func (a *ProxyApi) GetProxy(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

// PageProxy 分页查询代理节点。
func (a *ProxyApi) PageProxy(c *fiber.Ctx) error {
	var params *aiReq.ProxyPageParams
	err := a.BodyParser(c, &params, "代理分页")
	return response.Execute(c, a.service.Page, params, err)
}
