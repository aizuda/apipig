package api

import (
	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/app/ai/service"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"

	"github.com/gofiber/fiber/v2"
)

// AccessTokenTagApi 负责 API 密钥标签管理接口。
type AccessTokenTagApi struct {
	api.API
	service *service.AccessTokenTagService
}

func NewAccessTokenTagApi(tagService *service.AccessTokenTagService) *AccessTokenTagApi {
	return &AccessTokenTagApi{service: tagService}
}

// SaveAccessTokenTag 创建或更新 API 密钥标签。
func (a *AccessTokenTagApi) SaveAccessTokenTag(c *fiber.Ctx) error {
	tag := new(model.AccessTokenTag)
	err := a.BodyParserVerify(c, tag, "API 密钥标签")
	return response.Execute(c, a.service.Save, &aiReq.AccessTokenTagSaveParams{Ctx: c, Tag: tag}, err)
}

// DeleteAccessTokenTag 批量删除 API 密钥标签。
func (a *AccessTokenTagApi) DeleteAccessTokenTag(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除 API 密钥标签")
	return response.Execute(c, a.service.Delete, idsReq, err)
}

// ListAccessTokenTag 查询全部 API 密钥标签。
func (a *AccessTokenTagApi) ListAccessTokenTag(c *fiber.Ctx) error {
	return response.Execute(c, a.service.List, &request.Empty{}, nil)
}

// SortAccessTokenTag 保存 API 密钥标签排序。
func (a *AccessTokenTagApi) SortAccessTokenTag(c *fiber.Ctx) error {
	params := new(aiReq.AccessTokenTagSortParams)
	err := a.BodyParserVerify(c, params, "API 密钥标签排序")
	return response.Execute(c, a.service.Sort, params, err)
}
