package api

import (
	"apipig/app/apps/code-review/service"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

type WebhookApi struct{ service *service.WebhookService }

func (a *WebhookApi) Handle(c *fiber.Ctx) error {
	result, err := a.service.Handle(c.Params("projectKey"), c)
	if err != nil {
		return response.Failed(c, err.Error())
	}
	return c.Status(fiber.StatusAccepted).JSON(response.Response{Code: response.Success, Data: result, Msg: "WebHook 已接收"})
}
