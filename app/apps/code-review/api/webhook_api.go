package api

import (
	"apipig/app/apps/code-review/service"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

type WebhookApi struct{ service *service.WebhookService }

const maxWebhookBodyBytes = 1024 * 1024

func (a *WebhookApi) Handle(c *fiber.Ctx) error {
	if len(c.Body()) > maxWebhookBodyBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.Response{
			Code: response.Error,
			Msg:  "WebHook 请求体超过 1 MiB 限制",
		})
	}
	result, err := a.service.Handle(c.Params("projectKey"), c)
	if err != nil {
		return response.Failed(c, err.Error())
	}
	return c.Status(fiber.StatusAccepted).JSON(response.Response{Code: response.Success, Data: result, Msg: "WebHook 已接收"})
}
