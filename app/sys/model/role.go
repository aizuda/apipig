package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// Role 系统角色表
type Role struct {
	api.MODEL
	Name   string `gorm:"size:50;not null" json:"name,omitempty"`                   // 名称
	Alias  string `gorm:"size:50" json:"alias,omitempty"`                           // 别名
	Remark string `gorm:"size:255" json:"remark,omitempty"`                         // 备注
	Status uint   `gorm:"type:smallint;not null;default:1" json:"status,omitempty"` // 状态 1、正常 2、禁用
	Sort   uint   `gorm:"type:int;not null;default:0" json:"sort,omitempty"`        // 排序

}

func (Role) TableName() string {
	return "ap_role"
}

// RoleResource 系统角色资源表
type RoleResource struct {
	ID         snowflake.ID `gorm:"type:bigint;primaryKey" json:"id,omitempty" swaggertype:"string"` // 主键ID
	RoleId     snowflake.ID `gorm:"type:bigint;not null" json:"roleId" swaggertype:"string"`         // 角色ID
	ResourceId snowflake.ID `gorm:"type:bigint;not null" json:"resourceId" swaggertype:"string"`     // 资源ID
}

func (RoleResource) TableName() string {
	return "ap_role_resource"
}
