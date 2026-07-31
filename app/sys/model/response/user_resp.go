package response

import (
	"apipig/app/sys/model"
	"apipig/toolkit/snowflake"
)

type UserResp struct {
	model.User
	RoleIds       []snowflake.ID `json:"roleIds"`       // 角色ID集合
	DepartmentIds []snowflake.ID `json:"departmentIds"` // 部门ID集合
}
