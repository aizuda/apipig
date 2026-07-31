package request

import (
	aiModel "apipig/app/ai/model"
	coreReq "apipig/core/api/request"
	"github.com/gofiber/fiber/v2"
)

// ProviderSaveParams 封装供应商保存参数和审计上下文。
type ProviderSaveParams struct {
	Ctx      *fiber.Ctx        `json:"-"`        // Fiber 请求上下文，用于生成审计字段
	Provider *aiModel.Provider `json:"provider"` // 待保存的供应商数据
}

// ProviderPageParams 定义供应商分页查询条件。
type ProviderPageParams struct {
	coreReq.PageInfo
	Keyword  string `json:"keyword"`  // 名称、编码、协议或模型关键字
	Name     string `json:"name"`     // 供应商名称，模糊匹配
	Code     string `json:"code"`     // 供应商编码，模糊匹配
	Protocol string `json:"protocol"` // 协议类型，精确匹配
	Status   uint   `json:"status"`   // 状态：1 启用，2 禁用
}

// GetPageInfo 返回通用分页参数。
func (p *ProviderPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}
