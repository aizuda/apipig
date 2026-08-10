package api

import (
	"errors"
	"strings"

	wechatReq "apipig/app/apps/wechat-bot/model/request"
	"apipig/app/apps/wechat-bot/service"
	coreAPI "apipig/core/api"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

type BotApi struct {
	coreAPI.API
	service *service.BotService
}

func (a *BotApi) Page(c *fiber.Ctx) error {
	var params wechatReq.BotPageParams
	err := a.BodyParser(c, &params, "微信 Bot 分页")
	return response.Execute(c, a.service.Page, &params, err)
}

func (a *BotApi) List(c *fiber.Ctx) error {
	var params wechatReq.BotListParams
	err := a.BodyParser(c, &params, "微信 Bot 列表")
	return response.Execute(c, a.service.List, &params, err)
}

func (a *BotApi) StartBind(c *fiber.Ctx) error {
	var request wechatReq.BindStartRequest
	err := a.BodyParser(c, &request, "微信 Bot 扫码绑定")
	params := &wechatReq.BindStartParams{Ctx: c, Request: request}
	return response.Execute(c, a.service.StartBind, params, err)
}

func (a *BotApi) PollBind(c *fiber.Ctx) error {
	params := &wechatReq.BindStatusParams{SessionID: c.Params("sessionId")}
	return response.Execute(c, a.service.PollBind, params, nil)
}

func (a *BotApi) Rename(c *fiber.Ctx) error {
	var params wechatReq.RenameRequest
	err := a.BodyParser(c, &params, "微信 Bot 重命名")
	return response.Execute(c, a.service.Rename, &params, err)
}

func (a *BotApi) Reconnect(c *fiber.Ctx) error {
	var params wechatReq.BotIDRequest
	err := a.BodyParser(c, &params, "微信 Bot 重连")
	return response.Execute(c, a.service.Reconnect, &params, err)
}

func (a *BotApi) SetEnabled(c *fiber.Ctx) error {
	var params wechatReq.SetEnabledRequest
	err := a.BodyParser(c, &params, "微信 Bot 状态")
	return response.Execute(c, a.service.SetEnabled, &params, err)
}

func (a *BotApi) Delete(c *fiber.Ctx) error {
	var params wechatReq.BotIDRequest
	err := a.BodyParser(c, &params, "删除微信 Bot")
	return response.Execute(c, a.service.Delete, &params, err)
}

func (a *BotApi) Contacts(c *fiber.Ctx) error {
	botID, err := a.KeyParser(c, "botId")
	params := &wechatReq.ContactListParams{BotID: botID}
	return response.Execute(c, a.service.Contacts, params, err)
}

func (a *BotApi) Messages(c *fiber.Ctx) error {
	var params wechatReq.MessagePageParams
	err := a.BodyParser(c, &params, "微信 Bot 消息分页")
	return response.Execute(c, a.service.Messages, &params, err)
}

func (a *BotApi) Send(c *fiber.Ctx) error {
	var params wechatReq.SendMessageRequest
	err := a.BodyParser(c, &params, "发送微信 Bot 消息")
	return response.Execute(c, a.service.Send, &params, err)
}

func (a *BotApi) WebhookCredentials(c *fiber.Ctx) error {
	var request wechatReq.WebhookCredentialsRequest
	err := a.BodyParser(c, &request, "微信 Bot Webhook 凭据")
	params := &wechatReq.WebhookCredentialsParams{Ctx: c, Request: request}
	return response.Execute(c, a.service.WebhookCredentials, params, err)
}

const maxWebhookBodyBytes = 16 * 1024

func (a *BotApi) WebhookPush(c *fiber.Ctx) error {
	if len(c.Body()) > maxWebhookBodyBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.Response{
			Code: response.Error, Msg: "Webhook 请求体超过 16 KiB 限制",
		})
	}
	var request wechatReq.WebhookPushRequest
	if err := a.BodyParser(c, &request, "Webhook 推送"); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Response{Code: response.Error, Msg: err.Error()})
	}
	result, err := a.service.WebhookPush(&wechatReq.WebhookPushParams{
		WebhookKey: c.Params("webhookKey"),
		Timestamp:  strings.TrimSpace(c.Get("X-Webhook-Timestamp")),
		Signature:  strings.TrimSpace(c.Get("X-Webhook-Signature")),
		Request:    request,
	})
	if err != nil {
		status := fiber.StatusBadRequest
		if errors.Is(err, service.ErrWebhookUnauthorized) {
			status = fiber.StatusUnauthorized
		}
		return c.Status(status).JSON(response.Response{Code: response.Error, Msg: err.Error()})
	}
	return response.Ok(c, result)
}
