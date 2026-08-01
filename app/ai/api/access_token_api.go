package api

import (
	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/app/ai/service"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/middleware"

	"github.com/gofiber/fiber/v2"
)

// AccessTokenApi 负责 AI 访问令牌表的管理接口。
type AccessTokenApi struct {
	api.API
	service *service.AccessTokenService
}

func NewAccessTokenApi(accessTokenService *service.AccessTokenService) *AccessTokenApi {
	return &AccessTokenApi{service: accessTokenService}
}

// SaveAccessToken 创建或更新访问令牌。
func (a *AccessTokenApi) SaveAccessToken(c *fiber.Ctx) error {
	m := new(model.AccessToken)
	err := a.BodyParserVerify(c, m, "访问 Token")
	return response.Execute(c, a.service.Save, &aiReq.AccessTokenSaveParams{Ctx: c, AccessToken: m}, err)
}

// ChangeAccessTokenStatus 切换 API 密钥启用、禁用状态。
func (a *AccessTokenApi) ChangeAccessTokenStatus(c *fiber.Ctx) error {
	params := new(aiReq.StatusChangeParams)
	err := a.BodyParserVerify(c, params, "API 密钥状态")
	return response.Execute(c, a.service.ChangeStatus, params, err)
}

// DeleteAccessToken 批量删除访问令牌。
// UpdateAccessTokenTags updates only API token tag relations.
func (a *AccessTokenApi) UpdateAccessTokenTags(c *fiber.Ctx) error {
	params := new(aiReq.AccessTokenTagUpdateParams)
	err := a.BodyParserVerify(c, params, "API \u5bc6\u94a5\u6807\u7b7e")
	return response.Execute(c, a.service.UpdateTags, params, err)
}

func (a *AccessTokenApi) DeleteAccessToken(c *fiber.Ctx) error {
	var idsReq *request.IdsReq
	err := a.BodyParser(c, &idsReq, "删除访问 Token")
	return response.Execute(c, a.service.Delete, idsReq, err)
}

// GetAccessToken 根据 ID 查询访问令牌。
func (a *AccessTokenApi) GetAccessToken(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

// PageAccessToken 分页查询访问令牌。
func (a *AccessTokenApi) PageAccessToken(c *fiber.Ctx) error {
	var params *aiReq.AccessTokenPageParams
	err := a.BodyParser(c, &params, "访问 Token 分页")
	return response.Execute(c, a.service.Page, params, err)
}

// StatisticsAccessToken 按 API 密钥聚合指定时间范围内的 Token 用量。
func (a *AccessTokenApi) StatisticsAccessToken(c *fiber.Ctx) error {
	params := new(aiReq.AccessTokenStatisticsParams)
	err := a.BodyParser(c, params, "API 密钥统计")
	claims := middleware.GetTokenClaims(c)
	if err == nil && claims != nil && claims.LoginType == middleware.LoginTypeAPIToken {
		if validateErr := service.AiService.AccessTokenService.ValidateSession(claims.AccessTokenID, c.IP()); validateErr != nil {
			return response.Failed(c, validateErr.Error())
		}
		params.AccessTokenID = claims.AccessTokenID
		params.Keyword = ""
		params.TagID = 0
		params.Page = 1
		params.PageSize = 1
	}
	return response.Execute(c, a.service.Statistics, params, err)
}
