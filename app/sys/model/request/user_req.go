package request

import (
	"apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

type UserAssignSetParams struct {
	Ctx      *fiber.Ctx     `json:"-"`
	ID       snowflake.ID   `json:"id,omitempty" swaggertype:"string"` // 主键ID
	Username string         `json:"username" validate:"required"`      // 用户登录名
	Password string         `json:"password"`                          // 用户登录密码
	RealName string         `json:"realName"`                          // 真实名称
	NickName string         `json:"nickName"`                          // 昵称
	Avatar   string         `json:"avatar"`                            // 用户头像
	Sex      uint           `json:"sex"`                               // 性别 1、男 2、女
	Phone    string         `json:"phone"`                             // 手机号
	Email    string         `json:"email"`                             // 邮箱
	JobNum   string         `json:"jobNum"`                            // 工号
	RoleIds  []snowflake.ID `json:"roleIds"`                           // 角色ID集合
}

type LoginParams struct {
	UUID         string `json:"uuid"`
	Username     string `json:"username" validate:"required"`     // 用户名
	Password     string `json:"password" validate:"required"`     // MD5密码，32位小写
	CaptchaToken string `json:"captchaToken" validate:"required"` // 验证码票据
	CaptchaCode  string `json:"captchaCode" validate:"required"`  // 验证码内容
}

type TokenLoginParams struct {
	Token        string `json:"token" validate:"required"`
	CaptchaToken string `json:"captchaToken" validate:"required"`
	CaptchaCode  string `json:"captchaCode" validate:"required"`
}

type RefreshTokenParams struct {
	RefreshToken string `json:"refreshToken" validate:"required"` // 刷新票据
}

type ResetPasswordParams struct {
	Ids      []snowflake.ID `json:"ids,omitempty" swaggertype:"array,string" validate:"required"` // 父ID
	Password string         `json:"password" validate:"required"`                                 // 密码
}

type AssignRolesParams struct {
	UserIds []snowflake.ID `json:"userIds,omitempty" swaggertype:"array,string" validate:"required"` // 用户ID列表
	RoleIds []snowflake.ID `json:"roleIds,omitempty" swaggertype:"array,string" validate:"required"` // 角色ID列表
}

type AssignDepartmentsParams struct {
	UserIds       []snowflake.ID `json:"userIds,omitempty" swaggertype:"array,string" validate:"required"`       // 用户ID列表
	DepartmentIds []snowflake.ID `json:"departmentIds,omitempty" swaggertype:"array,string" validate:"required"` // 部门ID列表
}

type UserParams struct {
	Username string `json:"username"`
	Status   uint   `json:"status,string,int"`
}

type UserPageParams struct {
	request.PageInfo
	Username     string       `json:"username"`                                    // 账号
	RealName     string       `json:"realName"`                                    // 真实名称
	NickName     string       `json:"nickName"`                                    // 昵称
	Sex          uint         `json:"sex"`                                         // 性别 1、男 2、女
	Phone        string       `json:"phone"`                                       // 手机号
	Email        string       `json:"email"`                                       // 邮箱
	Status       uint         `json:"status"`                                      // 状态 1、正常 2、禁用
	JobNum       string       `json:"jobNum"`                                      // 工号
	DepartmentId snowflake.ID `json:"departmentId,omitempty" swaggertype:"string"` // 部门ID

}
