package request

import (
	"apipig/core/api/request"
	"apipig/toolkit/snowflake"
	"github.com/gofiber/fiber/v2"
)

type ResourceCreateParams struct {
	Ctx         *fiber.Ctx   `json:"-"`
	Pid         snowflake.ID `json:"pid,omitempty" swaggertype:"string" validate:"required"` // 上一级 ID
	Title       string       `json:"title" validate:"required"`                              // 名称
	Alias       string       `json:"alias"`                                                  // 别名
	Type        uint         `json:"type" validate:"required"`                               // 类型 1，菜单 2，iframe 3，外链 4，按钮
	Code        string       `json:"code"`                                                   // 编码
	Redirect    string       `json:"redirect"`                                               // 重定向
	Path        string       `json:"path" validate:"required"`                               // 文件路径
	Icon        string       `json:"icon"`                                                   // 图标
	Status      uint         `json:"status"`                                                 // 状态 1、正常 2、禁用
	Sort        uint         `json:"sort,omitempty"`                                         // 排序
	Component   string       `json:"component,omitempty"`                                    // 视图
	Color       string       `json:"color,omitempty"`                                        // 颜色
	Hidden      bool         `json:"hidden,omitempty"`                                       // 隐藏菜单
	ParentRoute string       `json:"parentRoute,omitempty"`                                  // 上级路由
	KeepAlive   bool         `json:"keepAlive,omitempty"`                                    // 保留查询参数
	Query       string       `json:"query,omitempty"`                                        // 查询携带参数
}

type ResourceParams struct {
	Title string `json:"title"` // 名称

}

type ResourcePageParams struct {
	request.PageInfo
	Title string `json:"title"` // 名称

}
