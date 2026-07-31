package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// User 系统用户表
type User struct {
	api.MODEL
	Username      string `gorm:"not null;size:30" json:"username"`                                // 账号
	Password      string `gorm:"not null;size:32" json:"-"`                                       // 密码
	RealName      string `gorm:"size:100" json:"realName,omitempty"`                              // 真实名称
	NickName      string `gorm:"size:100" json:"nickName,omitempty"`                              // 昵称
	Avatar        string `gorm:"size:200" json:"avatar,omitempty"`                                // 用户头像
	Sex           uint   `gorm:"type:smallint;not null;default:1" json:"sex,omitempty"`           // 性别 1、男 2、女
	Phone         string `gorm:"size:11" json:"phone,omitempty"`                                  // 手机号
	PhoneVerified uint   `gorm:"type:smallint;not null;default:1" json:"phoneVerified,omitempty"` // 手机号是否验证 1、否 2、是
	Email         string `gorm:"size:100" json:"email,omitempty"`                                 // 邮箱
	EmailVerified uint   `gorm:"type:smallint;not null;default:1" json:"emailVerified,omitempty"` // 邮箱是否验证 1、否 2、是
	Status        uint   `gorm:"type:smallint;not null;default:1" json:"status,omitempty"`        // 状态 1、正常 2、禁用
	JobNum        string `gorm:"size:100" json:"jobNum,omitempty"`                                // 工号
	LoginTime     int64  `gorm:"type:bigint" json:"loginTime,omitempty"`                          // 最后登录时间
	PwdTime       int64  `gorm:"type:bigint" json:"pwdTime,omitempty"`                            // 修改密码时间
}

func (User) TableName() string {
	return "ap_user"
}

// UserRole 系统用户角色表
type UserRole struct {
	ID     snowflake.ID `gorm:"type:bigint;primaryKey" json:"id,omitempty" swaggertype:"string"` // 主键ID
	UserId snowflake.ID `gorm:"type:bigint;not null" json:"userId" swaggertype:"string"`         // 用户ID
	RoleId snowflake.ID `gorm:"type:bigint;not null" json:"roleId" swaggertype:"string"`         // 角色ID
}

func (UserRole) TableName() string {
	return "ap_user_role"
}

// UserSession 系统用户会话表
type UserSession struct {
	ID        snowflake.ID `gorm:"type:bigint;primaryKey" json:"id,omitempty" swaggertype:"string"` // 主键ID
	CreatedAt int64        `gorm:"type:bigint;not null" json:"createdAt,omitempty"`                 // 创建时间
	UserId    snowflake.ID `gorm:"type:bigint;not null" json:"userId" swaggertype:"string"`         // 用户ID
	Username  string       `gorm:"size:50;not null" json:"username"`                                // 用户名称
	Sid       string       `gorm:"size:100;not null" json:"sid"`                                    // 会话 ID
	Browser   string       `gorm:"size:255;not null" json:"browser"`                                // 浏览器信息
	Ip        string       `gorm:"size:50;not null" json:"ip"`                                      // IP地址
	St        int64        `gorm:"type:bigint" json:"st,omitempty"`                                 // 会话时间
}

func (UserSession) TableName() string {
	return "ap_user_session"
}
