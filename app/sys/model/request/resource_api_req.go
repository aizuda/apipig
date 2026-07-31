package request

import (
	"apipig/toolkit/snowflake"
	"github.com/gofiber/fiber/v2"
)

type ResourceApiCreateParams struct {
	Ctx        *fiber.Ctx   `json:"-"`
	ResourceId snowflake.ID `json:"resourceId,omitempty" swaggertype:"string" validate:"required"` // 资源 ID
	Url        string       `json:"url,omitempty" validate:"required"`                             // 接口地址
	Method     string       `json:"method,omitempty" validate:"required"`                          // 请求方法 get post 等

}

type ResourceApiParams struct {
	ResourceId snowflake.ID `json:"resourceId,omitempty" swaggertype:"string" validate:"required"` // 资源 ID
	Url        string       `json:"url,omitempty"`                                                 // 接口地址

}
