package request

import (
	reviewModel "apipig/app/apps/code-review/model"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

type PushChannelRequest struct {
	ID      snowflake.ID      `json:"id" swaggertype:"string"`
	Type    string            `json:"type"`
	Name    string            `json:"name"`
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config"`
}

type PushChannelsSaveRequest struct {
	ProjectID snowflake.ID         `json:"projectId" swaggertype:"string"`
	Channels  []PushChannelRequest `json:"channels"`
}

type PushChannelTestRequest struct {
	ProjectID snowflake.ID       `json:"projectId" swaggertype:"string"`
	Channel   PushChannelRequest `json:"channel"`
}

type ProjectSaveParams struct {
	Ctx                 *fiber.Ctx
	Project             *reviewModel.Project
	WebhookSecret       string
	RepositoryToken     string
	RotateWebhookSecret bool
}

type ProjectSaveRequest struct {
	reviewModel.Project
	WebhookSecret       string `json:"webhookSecret"`
	RepositoryToken     string `json:"repositoryToken"`
	RotateWebhookSecret bool   `json:"rotateWebhookSecret"`
}

type ProjectPageParams struct {
	coreReq.PageInfo
	Keyword  string `json:"keyword"`
	Provider string `json:"provider"`
	Status   uint   `json:"status"`
}

func (p *ProjectPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}

type TaskPageParams struct {
	coreReq.PageInfo
	ProjectID snowflake.ID `json:"projectId" swaggertype:"string"`
	EventType string       `json:"eventType"`
	Status    string       `json:"status"`
	Keyword   string       `json:"keyword"`
}

func (p *TaskPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}

type RetryTaskParams struct {
	ID snowflake.ID `json:"id" swaggertype:"string"`
}
