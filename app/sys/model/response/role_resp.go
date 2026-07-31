package response

import (
	"apipig/app/sys/model"
	"apipig/toolkit/snowflake"
)

type RoleResp struct {
	model.Role
	ResourceIds []snowflake.ID `json:"resourceIds"` // 菜单权限ID集合
}
