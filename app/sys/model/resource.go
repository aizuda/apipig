package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// Resource 系统资源表
type Resource struct {
	api.MODEL
	Pid         snowflake.ID `gorm:"type:bigint;not null;default:1" json:"pid,omitempty" swaggertype:"string"` // 上一级 ID
	Title       string       `gorm:"size:50;not null" json:"title,omitempty"`                                  // 名称
	Alias       string       `gorm:"size:50" json:"alias,omitempty"`                                           // 别名
	Type        uint         `gorm:"type:smallint;not null;default:1" json:"type,omitempty"`                   // 类型 1，菜单 2，iframe 3，外链 4，按钮
	Code        string       `gorm:"size:100" json:"code,omitempty"`                                           // 编码
	Redirect    string       `gorm:"size:100" json:"redirect,omitempty"`                                       // 重定向
	Path        string       `gorm:"size:100" json:"path,omitempty"`                                           // 文件路径
	Icon        string       `gorm:"size:100" json:"icon,omitempty"`                                           // 图标
	Status      uint         `gorm:"type:smallint;not null;default:1" json:"status,omitempty"`                 // 状态 1、正常 2、禁用
	Sort        uint         `gorm:"type:int;not null;default:0" json:"sort,omitempty"`                        // 排序
	Component   string       `gorm:"size:255" json:"component,omitempty"`                                      // 视图
	Color       string       `gorm:"size:255" json:"color,omitempty"`                                          // 颜色
	Hidden      bool         `gorm:"type:boolean;not null;default:false" json:"hidden"`                        // 隐藏菜单
	ParentRoute string       `gorm:"size:255" json:"parentRoute,omitempty"`                                    // 上级路由
	KeepAlive   bool         `gorm:"type:boolean;not null;default:false" json:"keepAlive"`                     // 保留查询参数
	Query       string       `gorm:"size:255" json:"query,omitempty"`                                          // 查询携带参数
}

func (Resource) TableName() string {
	return "ap_resource"
}
