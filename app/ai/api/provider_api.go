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

// ProviderApi 负责 AI 供应商表的管理接口。
type ProviderApi struct {
	api.API
	service *service.ProviderService
}

func NewProviderApi(providerService *service.ProviderService) *ProviderApi {
	return &ProviderApi{service: providerService}
}

// SaveProvider 创建或更新供应商配置。
func (a *ProviderApi) SaveProvider(c *fiber.Ctx) error {
	m := new(model.Provider)
	err := a.BodyParserVerify(c, m, "供应商配置")
	return response.Execute(c, a.service.Save, &aiReq.ProviderSaveParams{Ctx: c, Provider: m}, err)
}

// ChangeProviderStatus 切换供应商启用、禁用状态。
func (a *ProviderApi) ChangeProviderStatus(c *fiber.Ctx) error {
	params := new(aiReq.StatusChangeParams)
	err := a.BodyParserVerify(c, params, "供应商状态")
	return response.Execute(c, a.service.ChangeStatus, params, err)
}

// DeleteProvider 批量删除供应商配置。
func (a *ProviderApi) DeleteProvider(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除供应商")
	return response.Execute(c, a.service.Delete, idsReq, err)
}

// GetProvider 根据 ID 查询供应商配置。
func (a *ProviderApi) GetProvider(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

// ListProvider 查询所有启用状态的供应商配置。
func (a *ProviderApi) ListProvider(c *fiber.Ctx) error {
	return response.Execute(c, a.service.List, &request.Empty{}, nil)
}

// PageProvider 分页查询供应商配置。
func (a *ProviderApi) PageProvider(c *fiber.Ctx) error {
	var params *aiReq.ProviderPageParams
	err := a.BodyParser(c, &params, "供应商分页")
	return response.Execute(c, a.service.Page, params, err)
}
