package middleware

import (
	"apipig/core/api/response"
	"apipig/core/cache"
	"apipig/global"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RequireResourceAccess 按菜单资源校验后台用户权限。
//
// AI 管理面已经有稳定的菜单资源与角色关联，因此直接按资源路径鉴权，不依赖尚未完整
// 初始化的旧接口权限表。API Token 会话的可访问路由已由 JWTAuth 严格限定，这里只放行
// 它的日志和统计请求，具体数据范围仍由对应 API 强制绑定到当前 Token。
func RequireResourceAccess(resourcePath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetTokenClaims(c)
		if claims == nil {
			return response.Failed(c, "无法获取登录信息")
		}
		if claims.LoginType == LoginTypeAPIToken {
			if !isAPITokenSessionAllowed(c) {
				return response.Failed(c, "API Token 登录只能查看该密钥自身数据")
			}
			return c.Next()
		}
		if claims.LoginType != LoginTypeAccount || global.DB == nil {
			return response.Failed(c, "权限不足")
		}

		const query = `SELECT COUNT(DISTINCT resource.id)
			FROM ap_resource resource
			JOIN ap_role_resource role_resource ON role_resource.resource_id = resource.id
			JOIN ap_role role ON role.id = role_resource.role_id
			JOIN ap_user_role user_role ON user_role.role_id = role.id
			JOIN ap_user user_account ON user_account.id = user_role.user_id
			WHERE resource.path = ? AND user_account.id = ?
				AND resource.status = 1 AND role.status = 1 AND user_account.status = 1
				AND resource.deleted_at = 0 AND role.deleted_at = 0 AND user_account.deleted_at = 0`
		var count int64
		if err := global.DB.Raw(query, resourcePath, claims.ID).Scan(&count).Error; err != nil || count == 0 {
			return response.Failed(c, "权限不足")
		}
		return c.Next()
	}
}

// RequireAIResourceAccess 将 AI 管理接口映射到对应菜单资源，避免同一前缀注册多个
// Fiber Use 中间件后发生权限叠加。
func RequireAIResourceAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		const marker = "/ai/gateway/"
		index := strings.Index(c.Path(), marker)
		if index < 0 {
			return response.Failed(c, "权限不足")
		}
		endpoint := strings.TrimPrefix(c.Path()[index+len(marker):], "/")
		resourcePath := ""
		switch {
		case endpoint == "summary" || endpoint == "chat" || endpoint == "chat/stream":
			resourcePath = "/ai-gateway/overview"
		case endpoint == "provider" || strings.HasPrefix(endpoint, "provider/"):
			resourcePath = "/ai-gateway/providers"
		case endpoint == "channel-account" || strings.HasPrefix(endpoint, "channel-account/"):
			resourcePath = "/ai-gateway/channel-accounts"
		case endpoint == "channel" || strings.HasPrefix(endpoint, "channel/"):
			resourcePath = "/ai-gateway/channels"
		case endpoint == "token" || strings.HasPrefix(endpoint, "token/"),
			endpoint == "token-tag" || strings.HasPrefix(endpoint, "token-tag/"):
			resourcePath = "/ai-gateway/tokens"
		case endpoint == "proxy" || strings.HasPrefix(endpoint, "proxy/"):
			resourcePath = "/ai-gateway/proxies"
		case endpoint == "log" || strings.HasPrefix(endpoint, "log/"):
			resourcePath = "/ai-gateway/logs"
		default:
			return response.Failed(c, "权限不足")
		}
		return RequireResourceAccess(resourcePath)(c)
	}
}

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
	if tc == nil {
		return "no"
	}
	key := fmt.Sprintf("%s_%s_%s", url, method, tc.ID)
	if cache.RbacCache != nil {
		_flag, err := cache.RbacCache.Get(key)
		if err == nil {
			return string(_flag)
		}
	}

	// 数据库查询验证权限
	sql := `SELECT COUNT(*) FROM apipig_resource_api WHERE url=? AND method=? AND resource_id IN (SELECT c.resource_id FROM apipig_role_resource c
JOIN apipig_role r ON c.role_id=r.id JOIN apipig_user_role u ON r.id=u.role_id WHERE u.user_id=?)`
	var ct int64
	if global.DB.Raw(sql, url, method, tc.ID).Count(&ct).Error != nil || ct == 0 {
		flag = "no"
	}
	if cache.RbacCache != nil {
		_ = cache.RbacCache.Set(key, []byte(flag))
	}
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
