package middleware

import (
	"apipig/core/cache"
	"apipig/global"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RbacHandler() fiber.Handler {
	// rbac 权限处理
	return func(c *fiber.Ctx) error {
		return c.Next()
		//if IsAllow(c) == "ok" {
		//	return c.Next()
		//}
		//return response.FailedDetailed(c, nil, "权限不足")
	}
}

var skipUrls = []string{"/v1/sys/user/info", "/v1/sys/resource/list-menu", "/v1/thing/device-post"}

func IsAllow(c *fiber.Ctx) string {
	flag := "ok"
	url := c.Path()
	if contains(skipUrls, url) {
		return flag
	}

	method := c.Method()
	tc := GetTokenClaims(c)
	key := fmt.Sprintf("%s_%s_%s", url, method, tc.ID)
	_flag, err := cache.RbacCache.Get(key)
	if err == nil {
		return string(_flag)
	}

	// 数据库查询验证权限
	sql := `SELECT COUNT(*) FROM apipig_resource_api WHERE url=? AND method=? AND resource_id IN (SELECT c.resource_id FROM apipig_role_resource c
JOIN apipig_role r ON c.role_id=r.id JOIN apipig_user_role u ON r.id=u.role_id WHERE u.user_id=?)`
	var ct int64
	if global.DB.Raw(sql, url, method, tc.ID).Count(&ct).Error != nil || ct == 0 {
		flag = "no"
	}
	_ = cache.RbacCache.Set(key, []byte(flag))
	return flag
}

func contains(slice []string, url string) bool {
	for _, elem := range slice {
		if strings.Contains(url, elem) {
			return true
		}
	}
	return false
}
