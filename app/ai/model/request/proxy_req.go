package request

import (
	aiModel "apipig/app/ai/model"
	coreReq "apipig/core/api/request"
	"github.com/gofiber/fiber/v2"
)

// ProxySaveParams 封装代理节点保存参数和审计上下文。
type ProxySaveParams struct {
	Ctx   *fiber.Ctx     `json:"-"`     // Fiber 请求上下文，用于生成审计字段
	Proxy *aiModel.Proxy `json:"proxy"` // 待保存的代理节点数据
}

// ProxyPageParams 定义代理节点分页查询条件。
type ProxyPageParams struct {
	coreReq.PageInfo
	Keyword string `json:"keyword"` // 名称、地址、区域或协议关键字
	Name    string `json:"name"`    // 代理名称，模糊匹配
	Region  string `json:"region"`  // 区域，模糊匹配
	Status  uint   `json:"status"`  // 状态：1 启用，2 禁用
}

// GetPageInfo 返回通用分页参数。
func (p *ProxyPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}
