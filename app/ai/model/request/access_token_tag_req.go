package request

import (
	aiModel "apipig/app/ai/model"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

// AccessTokenTagSaveParams 封装 API 密钥标签保存参数和审计上下文。
type AccessTokenTagSaveParams struct {
	Ctx *fiber.Ctx              `json:"-"`
	Tag *aiModel.AccessTokenTag `json:"tag"`
}

// AccessTokenTagSortParams 定义 API 密钥标签的完整排序结果。
type AccessTokenTagSortParams struct {
	IDs []snowflake.ID `json:"ids" validate:"required" swaggertype:"array,string"`
}
