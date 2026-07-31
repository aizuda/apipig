package request

import (
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"
)

// CallLogPageParams 定义调用日志分页查询条件。
type CallLogPageParams struct {
	coreReq.PageInfo
	Keyword    string       `json:"keyword"`                            // 请求、模型、路径、IP 或错误关键字
	Model      string       `json:"model"`                              // 模型名称，模糊匹配
	ProviderID snowflake.ID `json:"providerId" swaggertype:"string"`    // 供应商 ID
	ChannelID  snowflake.ID `json:"channelId" swaggertype:"string"`     // 渠道账号 ID
	TokenID    snowflake.ID `json:"accessTokenId" swaggertype:"string"` // 调用方访问令牌 ID
	Success    uint         `json:"success"`                            // 调用结果：1 成功，2 失败
	StartAt    int64        `json:"startAt"`                            // 开始时间，毫秒时间戳
	EndAt      int64        `json:"endAt"`                              // 结束时间，毫秒时间戳
}

// GetPageInfo 返回通用分页参数。
func (p *CallLogPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}
