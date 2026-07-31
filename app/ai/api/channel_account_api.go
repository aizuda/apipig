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

// ChannelAccountApi 负责渠道注册账户管理接口。
type ChannelAccountApi struct {
	api.API
	service *service.ChannelAccountService
}

func NewChannelAccountApi(channelAccountService *service.ChannelAccountService) *ChannelAccountApi {
	return &ChannelAccountApi{service: channelAccountService}
}

func (a *ChannelAccountApi) SaveChannelAccount(c *fiber.Ctx) error {
	m := new(model.ChannelAccount)
	err := a.BodyParserVerify(c, m, "渠道账户")
	return response.Execute(c, a.service.Save, &aiReq.ChannelAccountSaveParams{Ctx: c, Account: m}, err)
}

func (a *ChannelAccountApi) ChangeChannelAccountStatus(c *fiber.Ctx) error {
	params := new(aiReq.StatusChangeParams)
	err := a.BodyParserVerify(c, params, "渠道账户状态")
	return response.Execute(c, a.service.ChangeStatus, params, err)
}

func (a *ChannelAccountApi) DeleteChannelAccount(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除渠道账户")
	return response.Execute(c, a.service.Delete, idsReq, err)
}

func (a *ChannelAccountApi) GetChannelAccount(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

func (a *ChannelAccountApi) PageChannelAccount(c *fiber.Ctx) error {
	var params *aiReq.ChannelAccountPageParams
	err := a.BodyParser(c, &params, "渠道账户分页")
	return response.Execute(c, a.service.Page, params, err)
}
