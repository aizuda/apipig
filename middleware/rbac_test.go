package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sysModel "apipig/app/sys/model"
	coreAPI "apipig/core/api"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRequireResourceAccess(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:resource-access?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&sysModel.User{}, &sysModel.Role{}, &sysModel.Resource{},
		&sysModel.UserRole{}, &sysModel.RoleResource{},
	))
	userID, roleID, resourceID := snowflake.ID(101), snowflake.ID(102), snowflake.ID(103)
	require.NoError(t, database.Create(&sysModel.User{
		MODEL: coreAPI.MODEL{ID: userID}, Username: "authorized", Password: "x", Status: 1,
	}).Error)
	require.NoError(t, database.Create(&sysModel.Role{
		MODEL: coreAPI.MODEL{ID: roleID}, Name: "AI 管理员", Status: 1,
	}).Error)
	require.NoError(t, database.Create(&sysModel.Resource{
		MODEL: coreAPI.MODEL{ID: resourceID}, Title: "供应商", Path: "/ai-gateway/providers", Status: 1,
	}).Error)
	require.NoError(t, database.Create(&sysModel.UserRole{ID: 104, UserId: userID, RoleId: roleID}).Error)
	require.NoError(t, database.Create(&sysModel.RoleResource{ID: 105, RoleId: roleID, ResourceId: resourceID}).Error)

	previousDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previousDB })

	t.Run("有资源权限的后台用户可访问", func(t *testing.T) {
		response := testResourceAccess(t, &TokenClaims{ID: userID, LoginType: LoginTypeAccount})
		require.Equal(t, fiber.StatusNoContent, response.StatusCode)
	})
	t.Run("没有角色关联的后台用户被拒绝", func(t *testing.T) {
		response := testResourceAccess(t, &TokenClaims{ID: 999, LoginType: LoginTypeAccount})
		require.Equal(t, fiber.StatusOK, response.StatusCode)
	})
	t.Run("API Token 会话交由 JWT 路由白名单继续约束", func(t *testing.T) {
		response := testResourceAccess(t, &TokenClaims{ID: 999, LoginType: LoginTypeAPIToken})
		require.Equal(t, fiber.StatusNoContent, response.StatusCode)
	})
	t.Run("AI 路由只校验自身对应的资源", func(t *testing.T) {
		allowed := testAIResourceAccess(t, userID, "/v1/ai/gateway/provider/page")
		require.Equal(t, fiber.StatusNoContent, allowed.StatusCode)
		denied := testAIResourceAccess(t, userID, "/v1/ai/gateway/channel/page")
		require.Equal(t, fiber.StatusOK, denied.StatusCode)
	})
}

func testAIResourceAccess(t *testing.T, userID snowflake.ID, path string) *http.Response {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("tokenClaims", &TokenClaims{ID: userID, LoginType: LoginTypeAccount})
		return c.Next()
	})
	app.Use(RequireAIResourceAccess())
	app.Post("/v1/ai/gateway/*", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	response, err := app.Test(httptest.NewRequest(http.MethodPost, path, nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func testResourceAccess(t *testing.T, claims *TokenClaims) *http.Response {
	t.Helper()
	app := fiber.New()
	app.Post("/v1/ai/gateway/log/page", func(c *fiber.Ctx) error {
		c.Locals("tokenClaims", claims)
		return c.Next()
	}, RequireResourceAccess("/ai-gateway/providers"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	response, err := app.Test(httptest.NewRequest(http.MethodPost, "/v1/ai/gateway/log/page", nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}
