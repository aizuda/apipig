package api

import (
	sysReq "apipig/app/sys/model/request"
	"apipig/app/sys/service"
	"apipig/core/api"
	"apipig/core/api/response"
	"fmt"
	"github.com/gofiber/fiber/v2"
)

type WebApi struct {
	api.API
}

var webService = service.SysService.WebService

// Captcha
// @Tags Web
// @Summary 验证码
// @Router /v1/captcha [get]
func (a *WebApi) Captcha(c *fiber.Ctx) error {
	return webService.Captcha(c)
}

// PublicKey
// @Tags Web
// @Summary 获取登录公钥
// @Produce json
// @Success 200 {string} string "{\"success\":true,\"data\":{\"uuid\":\"\",\"publicKey\":\"\"}}"
// @Router /v1/public-key [post]
func (a *WebApi) PublicKey(c *fiber.Ctx) error {
	publicKey, err := webService.PublicKey()
	if err != nil {
		return response.Failed(c, err.Error())
	}
	return response.Ok(c, publicKey)
}

// Login
// @Tags Web
// @Summary 用户登录
// @accept json
// @Produce json
// @Param data body sysReq.LoginParams true "用户名, 密码, 验证码"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"登陆成功"}"
// @Router /v1/login [post]
func (a *WebApi) Login(c *fiber.Ctx) error {
	params := new(sysReq.LoginParams)
	err := a.BodyParserVerify(c, params, "用户登录")
	login, err := webService.Login(c, params)
	if err != nil {
		return response.Failed(c, err.Error())
	}
	return response.Ok(c, login)
}

// RefreshToken
// @Tags Web
// @Summary 刷新Token
// @accept json
// @Produce json
// @Param data body sysReq.RefreshTokenParams true "刷新票据"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"刷新Token成功"}"
// @Router /v1/refresh-token [post]
// AuthorizeToken 使用 API 密钥签发仅限用量页面的临时授权凭证。
func (a *WebApi) AuthorizeToken(c *fiber.Ctx) error {
	params := new(sysReq.TokenAuthorizationParams)
	if err := a.BodyParserVerify(c, params, "API Token authorization"); err != nil {
		return response.Failed(c, err.Error())
	}
	login, err := webService.AuthorizeToken(c, params)
	if err != nil {
		return response.Failed(c, err.Error())
	}
	return response.Ok(c, login)
}

func (a *WebApi) RefreshToken(c *fiber.Ctx) error {
	params := new(sysReq.RefreshTokenParams)
	err := a.BodyParserVerify(c, params, "刷新票据")
	if err == nil {
		loginResp, err := webService.RefreshToken(c, params)
		if err == nil {
			return response.Ok(c, loginResp)
		}
	}
	return response.Result(c, response.ExpiredRefreshToken, nil, "长时间未操作超时")
}

// TestHttpPush
// @Tags Web
// @Summary 测试http推送
// @accept json
// @Produce json
// @Success 200 {string} string "{"success":true,"data":{},"msg":"刷新Token成功"}"
// @Router /v1/test-http-push [post]
func (a *WebApi) TestHttpPush(c *fiber.Ctx) error {
	fmt.Println("推送内容：" + string(c.BodyRaw()))
	return response.Ok(c, "ok")
}
