package api

import (
	reviewReq "apipig/app/apps/code-review/model/request"
	"apipig/app/apps/code-review/service"
	"apipig/core/api"
	"apipig/core/api/response"
	"github.com/gofiber/fiber/v2"
)

type PushChannelApi struct {
	api.API
	service *service.PushChannelFacade
}

func (a *PushChannelApi) Save(c *fiber.Ctx) error {
	var params reviewReq.PushChannelsSaveRequest
	err := a.BodyParser(c, &params, "代码评审推送渠道")
	return response.Execute(c, a.service.Save, &params, err)
}

func (a *PushChannelApi) Test(c *fiber.Ctx) error {
	var params reviewReq.PushChannelTestRequest
	err := a.BodyParser(c, &params, "测试代码评审推送渠道")
	return response.Execute(c, a.service.Test, &params, err)
}
