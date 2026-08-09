package router

import (
	"apipig/app/apps/code-review/api"

	"github.com/gofiber/fiber/v2"
)

type CodeReviewRouter struct{}

func (r *CodeReviewRouter) InitWebhookRouter(router fiber.Router) {
	router.Post("/apps/code-review/webhook/:projectKey", api.ReviewApi.WebhookApi.Handle)
}

func (r *CodeReviewRouter) InitAdminRouter(router fiber.Router) {
	project := router.Group("project/")
	project.Post("save", api.ReviewApi.ProjectApi.Save)
	project.Get("get", api.ReviewApi.ProjectApi.Get)
	project.Post("page", api.ReviewApi.ProjectApi.Page)
	project.Post("delete", api.ReviewApi.ProjectApi.Delete)
	project.Get("push-channels", api.ReviewApi.PushChannelApi.ListForEdit)
	task := router.Group("task/")
	task.Post("page", api.ReviewApi.TaskApi.Page)
	task.Get("get", api.ReviewApi.TaskApi.Get)
	task.Post("retry", api.ReviewApi.TaskApi.Retry)
	project.Post("push-channels", api.ReviewApi.PushChannelApi.Save)
	project.Post("push-channels/test", api.ReviewApi.PushChannelApi.Test)
}
