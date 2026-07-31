package api

import (
	reviewReq "apipig/app/apps/code-review/model/request"
	"apipig/app/apps/code-review/service"
	"apipig/core/api"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

type TaskApi struct {
	api.API
	service *service.TaskService
}

func (a *TaskApi) Page(c *fiber.Ctx) error {
	var params reviewReq.TaskPageParams
	err := a.BodyParser(c, &params, "代码评审任务分页")
	return response.Execute(c, a.service.Page, &params, err)
}
func (a *TaskApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}
func (a *TaskApi) Retry(c *fiber.Ctx) error {
	var params reviewReq.RetryTaskParams
	err := a.BodyParser(c, &params, "重试代码评审任务")
	return response.Execute(c, a.service.Retry, &params, err)
}
