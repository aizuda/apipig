package request

import "apipig/toolkit/snowflake"

// StatusChangeParams 定义 AI 管理资源的启用、禁用状态切换参数。
type StatusChangeParams struct {
	ID     snowflake.ID `json:"id" swaggertype:"string" validate:"required"` // 资源 ID
	Status uint         `json:"status" validate:"required"`                  // 状态：1 启用，2 禁用
}
