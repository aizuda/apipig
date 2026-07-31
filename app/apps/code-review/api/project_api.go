package api

import (
	reviewReq "apipig/app/apps/code-review/model/request"
	"apipig/app/apps/code-review/service"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

type ProjectApi struct {
	api.API
	service *service.ProjectService
}

func (a *ProjectApi) Save(c *fiber.Ctx) error {
	var body reviewReq.ProjectSaveRequest
	err := a.BodyParser(c, &body, "代码评审项目")
	params := &reviewReq.ProjectSaveParams{Ctx: c, Project: &body.Project, WebhookSecret: body.WebhookSecret, RepositoryToken: body.RepositoryToken, RotateWebhookSecret: body.RotateWebhookSecret}
	return response.Execute(c, a.service.Save, params, err)
}
func (a *ProjectApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}
func (a *ProjectApi) Page(c *fiber.Ctx) error {
	var params reviewReq.ProjectPageParams
	err := a.BodyParser(c, &params, "代码评审项目分页")
	return response.Execute(c, a.service.Page, &params, err)
}
func (a *ProjectApi) Delete(c *fiber.Ctx) error {
	var params request.IdsReq
	err := a.BodyParser(c, &params, "删除代码评审项目")
	return response.Execute(c, a.service.Delete, params.Ids, err)
}
