package api

import "apipig/app/apps/code-review/service"

type ReviewApiGroup struct {
	ProjectApi *ProjectApi
	TaskApi    *TaskApi
	WebhookApi *WebhookApi
}

func NewReviewApiGroup(services *service.ReviewServiceGroup) *ReviewApiGroup {
	return &ReviewApiGroup{ProjectApi: &ProjectApi{service: services.ProjectService}, TaskApi: &TaskApi{service: services.TaskService}, WebhookApi: &WebhookApi{service: services.WebhookService}}
}

var ReviewApi = NewReviewApiGroup(service.ReviewService)
