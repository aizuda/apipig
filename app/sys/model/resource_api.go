package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// ResourceApi 系统资源接口表
type ResourceApi struct {
	api.MODEL
	ResourceId snowflake.ID `gorm:"type:bigint;not null" json:"resource_id,omitempty" swaggertype:"string"` // 资源 ID
	Url        string       `gorm:"size:255;not null" json:"url,omitempty"`                                 // 接口地址
	Method     string       `gorm:"size:30;not null" json:"method,omitempty"`                               // 请求方法 get post 等
}

func (ResourceApi) TableName() string {
	return "ap_resource_api"
}
