package response

import (
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Code string      `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

const (
	Error               = "0" // 失败
	Success             = "1" // 成功
	ExpiredToken        = "2" // 票据过期
	ExpiredRefreshToken = "3" // 刷新票据过期
)

func Result(c *fiber.Ctx, code string, data interface{}, msg string) error {
	// 开始时间
	return c.JSON(Response{
		code,
		data,
		msg,
	})
}

func Ok(c *fiber.Ctx, data interface{}) error {
	return OkDetailed(c, data, "操作成功")
}

func OkDetailed(c *fiber.Ctx, data interface{}, message string) error {
	return Result(c, Success, data, message)
}

func Failed(c *fiber.Ctx, message string) error {
	return FailedDetailed(c, nil, message)
}

func FailedDetailed(c *fiber.Ctx, data interface{}, message string) error {
	return Result(c, Error, data, message)
}

func FailedExpiredToken(c *fiber.Ctx) error {
	return Result(c, ExpiredToken, nil, "the token has expired")
}

func Execute[T any, R any](c *fiber.Ctx, callback func(params T) (R, error), params T, err error) error {
	if err == nil {
		var data any
		data, err = callback(params)
		if err == nil {
			return Ok(c, data)
		}
	}
	return Failed(c, err.Error())
}

func ParseResponse(jsonStr string) (*Response, error) {
	var resp *Response
	err := json.Unmarshal([]byte(jsonStr), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != Success {
		return nil, errors.New(resp.Msg)
	}
	return resp, nil
}
