package request

import (
    "apipig/core/api/request"
    "apipig/toolkit/snowflake"

    "github.com/gofiber/fiber/v2"
)

type RoleResourceSetParams struct {
    Ctx         *fiber.Ctx     `json:"-"`
    ID          snowflake.ID   `json:"id,omitempty" swaggertype:"string"` // 主键ID
    Name        string         `json:"name" validate:"required"`          // 名称
    Alias       string         `json:"alias" validate:"required"`         // 别名
    Remark      string         `json:"remark"`                            // 备注
    Status      uint           `json:"status"`                            // 状态 1、正常 2、禁用
    Sort        uint           `json:"sort"`                              // 排序
    ResourceIds []snowflake.ID `json:"resourceIds"`                       // 菜单权限ID集合
}

type RoleParams struct {
    Name string `json:"name"` // 名称

}

type RolePageParams struct {
    request.PageInfo
    Name   string `json:"name"`   // 名称
    Alias  string `json:"alias"`  // 别名
    Status uint   `json:"status"` // 状态 1、正常 2、禁用

}
