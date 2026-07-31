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

// ChannelApi 负责 AI 渠道账号表的管理接口。
type ChannelApi struct {
	api.API
	service *service.ChannelService
}

func NewChannelApi(channelService *service.ChannelService) *ChannelApi {
	return &ChannelApi{service: channelService}
}

// SaveChannel 创建或更新渠道账号。
func (a *ChannelApi) SaveChannel(c *fiber.Ctx) error {
	m := new(model.Channel)
	err := a.BodyParserVerify(c, m, "渠道账号")
	return response.Execute(c, a.service.Save, &aiReq.ChannelSaveParams{Ctx: c, Channel: m}, err)
}

// ChangeChannelStatus 切换渠道账号启用、禁用状态。
func (a *ChannelApi) ChangeChannelStatus(c *fiber.Ctx) error {
	params := new(aiReq.StatusChangeParams)
	err := a.BodyParserVerify(c, params, "渠道状态")
	return response.Execute(c, a.service.ChangeStatus, params, err)
}

// DeleteChannel 批量删除渠道账号。
func (a *ChannelApi) DeleteChannel(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除渠道")
	return response.Execute(c, a.service.Delete, idsReq, err)
}

// GetChannel 根据 ID 查询渠道账号。
func (a *ChannelApi) GetChannel(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

// PageChannel 分页查询渠道账号。
func (a *ChannelApi) PageChannel(c *fiber.Ctx) error {
	var params *aiReq.ChannelPageParams
	err := a.BodyParser(c, &params, "渠道分页")
	return response.Execute(c, a.service.Page, params, err)
}
