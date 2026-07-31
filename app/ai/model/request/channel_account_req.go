package request

import (
	aiModel "apipig/app/ai/model"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

// ChannelAccountSaveParams 封装渠道账户保存参数和审计上下文。
type ChannelAccountSaveParams struct {
	Ctx     *fiber.Ctx              `json:"-"`
	Account *aiModel.ChannelAccount `json:"account"`
}

// ChannelAccountPageParams 定义渠道账户分页查询条件。
type ChannelAccountPageParams struct {
	coreReq.PageInfo
	ChannelID snowflake.ID `json:"channelId" swaggertype:"string"`
	Keyword   string       `json:"keyword"`
	Name      string       `json:"name"`
	Status    uint         `json:"status"`
}

// GetPageInfo 返回通用分页参数。
func (p *ChannelAccountPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}
