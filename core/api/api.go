package api

import (
	"apipig/toolkit/snowflake"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/validate"
	"github.com/gookit/validate/locales/zhcn"
)

type API struct {
}

func init() {
	// 本地化规则只需注册一次，避免每个请求重复修改全局校验器状态。
	zhcn.RegisterGlobal()
}

func (a *API) IdParser(c *fiber.Ctx) (snowflake.ID, error) {
	id, err := snowflake.ParseString(c.Query("id"))
	if err != nil {
		err = errors.New("参数ID解析失败")
	}
	return id, err
}

func (a *API) KeyParser(c *fiber.Ctx, key string) (snowflake.ID, error) {
	id, err := snowflake.ParseString(c.Query(key))
	if err != nil {
		err = fmt.Errorf("参数%s解析失败", key)
	}
	return id, err
}

func (a *API) BodyParser(c *fiber.Ctx, params interface{}, message string) error {
	if err := c.BodyParser(params); err != nil {
		return errors.New(message + "参数解析失败")
	}
	return nil
}

func (a *API) BodyParserIdVerify(c *fiber.Ctx, params interface{}, message string) error {
	return a.BodyParserVerifyAddRules(c, params, func(v *validate.Validation) {
		v.AddRule("ID", "required")
	}, message)
}

func (a *API) BodyParserVerify(c *fiber.Ctx, params interface{}, message string) error {
	return a.BodyParserVerifyAddRules(c, params, nil, message)
}

func (a *API) BodyParserVerifyAddRules(c *fiber.Ctx, params interface{}, addRules func(v *validate.Validation), message string) error {
	if err := a.BodyParser(c, params, message); err != nil {
		return err
	}

	// 表单验证 https://gookit.github.io/validate/#/README.zh-CN
	v := validate.Struct(params)
	if addRules != nil {
		addRules(v)
	}
	if v != nil {
		if !v.Validate() {
			return v.Errors.OneError()
		}
	}
	return nil
}
