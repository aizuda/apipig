package response

import (
	"apipig/app/sys/model"
)

type ResourceMenuResp struct {
	Menus       []ResourceMetaResp `json:"menus"`       // 菜单
	Permissions []string           `json:"permissions"` // 权限
}

type ResourceMetaResp struct {
	Name      string             `json:"name"`               // 名称
	Redirect  string             `json:"redirect,omitempty"` // 重定向
	Path      string             `json:"path"`               // 文件路径
	Component string             `json:"component"`          // 视图
	Meta      map[string]any     `json:"meta"`               // 其它元素
	Children  []ResourceMetaResp `json:"children,omitempty"` // 子资源
}

type ResourceTreeResp struct {
	model.Resource
	Children []ResourceTreeResp `json:"children,omitempty"` // 子资源
}
