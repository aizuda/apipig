package api

import (
	aiReq "apipig/app/ai/model/request"
	"apipig/app/ai/service"
	"apipig/core/api"
	"apipig/core/api/response"
	"apipig/middleware"

	"github.com/gofiber/fiber/v2"
)

type CallLogApi struct {
	api.API
	service *service.CallLogService
}

func NewCallLogApi(callLogService *service.CallLogService) *CallLogApi {
	return &CallLogApi{service: callLogService}
}

func (a *CallLogApi) PageCallLog(c *fiber.Ctx) error {
	var params *aiReq.CallLogPageParams
	err := a.BodyParser(c, &params, "call log page")
	claims := middleware.GetTokenClaims(c)
	if err == nil && claims != nil && claims.LoginType == middleware.LoginTypeAPIToken {
		if validateErr := service.AiService.AccessTokenService.ValidateSession(claims.AccessTokenID, c.IP()); validateErr != nil {
			return response.Failed(c, validateErr.Error())
		}
		if params == nil {
			params = new(aiReq.CallLogPageParams)
		}
		params.TokenID = claims.AccessTokenID
		return response.Execute(c, a.service.PageForAccessToken, params, nil)
	}
	return response.Execute(c, a.service.Page, params, err)
}
