package request

import (
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

type BotPageParams struct {
	coreReq.PageInfo
	Keyword string `json:"keyword"`
	Status  string `json:"status"`
}

func (p *BotPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 20}
	}
	return p.PageInfo
}

type BindStartRequest struct {
	Name  string       `json:"name"`
	BotID snowflake.ID `json:"botId" swaggertype:"string"`
}

type BindStartParams struct {
	Ctx     *fiber.Ctx
	Request BindStartRequest
}

type BindStatusParams struct {
	SessionID string
}

type RenameRequest struct {
	ID   snowflake.ID `json:"id" swaggertype:"string"`
	Name string       `json:"name"`
}

type BotIDRequest struct {
	ID snowflake.ID `json:"id" swaggertype:"string"`
}

type SetEnabledRequest struct {
	ID      snowflake.ID `json:"id" swaggertype:"string"`
	Enabled bool         `json:"enabled"`
}

type ContactListParams struct {
	BotID snowflake.ID
}

type MessagePageParams struct {
	coreReq.PageInfo
	BotID  snowflake.ID `json:"botId" swaggertype:"string"`
	UserID string       `json:"userId"`
}

func (p *MessagePageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 100}
	}
	return p.PageInfo
}

type SendMessageRequest struct {
	BotID   snowflake.ID `json:"botId" swaggertype:"string"`
	UserID  string       `json:"userId"`
	Content string       `json:"content"`
}
