package request

import (
	aiModel "apipig/app/ai/model"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"
	"github.com/gofiber/fiber/v2"
)

// AccessTokenSaveParams 封装访问令牌保存参数和审计上下文。
type AccessTokenSaveParams struct {
	Ctx         *fiber.Ctx           `json:"-"`           // Fiber 请求上下文，用于生成审计字段
	AccessToken *aiModel.AccessToken `json:"accessToken"` // 待保存的访问令牌数据
}

// AccessTokenTagUpdateParams 定义仅更新 API 密钥标签的请求。
type AccessTokenTagUpdateParams struct {
	ID     snowflake.ID   `json:"id" validate:"required"`
	TagIDs []snowflake.ID `json:"tagIds"`
}

// AccessTokenPageParams 定义访问令牌分页查询条件。
type AccessTokenPageParams struct {
	coreReq.PageInfo
	Keyword string `json:"keyword"` // 名称、模型或备注关键字
	Name    string `json:"name"`    // 令牌名称，模糊匹配
	Status  uint   `json:"status"`  // 状态：1 启用，2 禁用
}

// AccessTokenStatisticsParams 定义 API 密钥用量统计的时间范围；零值表示全部时间。
type AccessTokenStatisticsParams struct {
	coreReq.PageInfo
	Keyword       string       `json:"keyword"`
	TagID         snowflake.ID `json:"tagId" swaggertype:"string"`
	AccessTokenID snowflake.ID `json:"-"` // 仅由授权接口根据 JWT 注入，客户端不可指定
	StartAt       int64        `json:"startAt"`
	EndAt         int64        `json:"endAt"`
}

// GetPageInfo 返回通用分页参数。
func (p *AccessTokenPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}
