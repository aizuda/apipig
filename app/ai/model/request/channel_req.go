package request

import (
	aiModel "apipig/app/ai/model"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"
	"github.com/gofiber/fiber/v2"
)

// ChannelSaveParams 封装渠道账号保存参数和审计上下文。
type ChannelSaveParams struct {
	Ctx     *fiber.Ctx       `json:"-"`       // Fiber 请求上下文，用于生成审计字段
	Channel *aiModel.Channel `json:"channel"` // 待保存的渠道账号数据
}

// ChannelPageParams 定义渠道账号分页查询条件。
type ChannelPageParams struct {
	coreReq.PageInfo
	ProviderID snowflake.ID `json:"providerId" swaggertype:"string"` // 供应商 ID
	Keyword    string       `json:"keyword"`                         // 名称或模型关键字
	Name       string       `json:"name"`                            // 渠道名称，模糊匹配
	Model      string       `json:"model"`                           // 模型名称，模糊匹配
	Status     uint         `json:"status"`                          // 状态：1 启用，2 禁用
}

// GetPageInfo 返回通用分页参数。
func (p *ChannelPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}
