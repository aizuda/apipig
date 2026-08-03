package initialize

import (
	"fmt"

	sysModel "apipig/app/sys/model"
	"apipig/core/api"
	"apipig/toolkit"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminAccount struct {
	Username string
	Password string
}

const (
	adminRoleID snowflake.ID = 1
	adminUserID snowflake.ID = 1

	dashboardResourceID       snowflake.ID = 10001
	aiGatewayResourceID       snowflake.ID = 10002
	aiOverviewResourceID      snowflake.ID = 10003
	aiProvidersResourceID     snowflake.ID = 10004
	aiChannelsResourceID      snowflake.ID = 10005
	aiAccountsResourceID      snowflake.ID = 10006
	aiTokensResourceID        snowflake.ID = 10007
	aiProxiesResourceID       snowflake.ID = 10008
	aiLogsResourceID          snowflake.ID = 10009
	settingsResourceID        snowflake.ID = 10010
	settingsUsersResourceID   snowflake.ID = 10011
	settingsRolesResourceID   snowflake.ID = 10012
	settingsMenusResourceID   snowflake.ID = 10013
	settingsAccountResourceID snowflake.ID = 10014
	settingsMessageResourceID snowflake.ID = 10015
	codeReviewResourceID      snowflake.ID = 10016
	aiApplicationsResourceID  snowflake.ID = 10017
	remoteAgentResourceID     snowflake.ID = 10018

	seedCreatedAt      int64        = 1781936428406
	roleResourceIDBase int64        = 11000
	rootResourceID     snowflake.ID = 1
)

func initData(db *gorm.DB, admin *AdminAccount) error {
	if err := autoMigrate(db); err != nil {
		return err
	}

	data := getInitData(admin)
	if err := validateCurrentMenuResources(data.Resources); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := createIfNotExists(tx, data.Users); err != nil {
			return err
		}
		if err := createIfNotExists(tx, data.Roles); err != nil {
			return err
		}
		if err := upsertSeedResources(tx, data.Resources); err != nil {
			return err
		}
		if err := createIfNotExists(tx, data.ResourceApis); err != nil {
			return err
		}
		if err := syncAdminUserRoles(tx, data.UserRoles); err != nil {
			return err
		}
		if err := syncAdminRoleResources(tx, data.Resources, data.RoleResources); err != nil {
			return err
		}
		if admin == nil {
			return nil
		}
		return tx.Model(&sysModel.User{}).Where("id = ?", adminUserID).Updates(map[string]any{
			"username": admin.Username,
			"password": toolkit.GetPassword(admin.Password, admin.Username),
		}).Error
	})
}

type initDataSet struct {
	Users         []sysModel.User
	Roles         []sysModel.Role
	Resources     []sysModel.Resource
	ResourceApis  []sysModel.ResourceApi
	UserRoles     []sysModel.UserRole
	RoleResources []sysModel.RoleResource
}

func createIfNotExists[T any](db *gorm.DB, values []T) error {
	if len(values) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&values).Error
}

func upsertSeedResources(db *gorm.DB, resources []sysModel.Resource) error {
	if len(resources) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"pid", "title", "alias", "type", "code", "redirect", "path", "icon", "status", "sort",
			"component", "color", "hidden", "parent_route", "keep_alive", "query", "deleted_at", "updated_at",
		}),
	}).Create(&resources).Error
}

func syncAdminUserRoles(db *gorm.DB, userRoles []sysModel.UserRole) error {
	if len(userRoles) == 0 {
		return nil
	}
	for _, item := range userRoles {
		if err := db.Where("id = ? OR (user_id = ? AND role_id = ?)", item.ID, item.UserId, item.RoleId).Delete(&sysModel.UserRole{}).Error; err != nil {
			return err
		}
	}
	return db.Create(&userRoles).Error
}

func syncAdminRoleResources(db *gorm.DB, resources []sysModel.Resource, roleResources []sysModel.RoleResource) error {
	if len(roleResources) == 0 {
		return nil
	}
	resourcePaths := make([]string, 0, len(resources))
	for _, item := range resources {
		resourcePaths = append(resourcePaths, item.Path)
	}
	resourceIDs := db.Model(&sysModel.Resource{}).Select("id").Where("path IN ?", resourcePaths)
	if err := db.Where("role_id = ? AND resource_id IN (?)", adminRoleID, resourceIDs).Delete(&sysModel.RoleResource{}).Error; err != nil {
		return err
	}
	return db.Create(&roleResources).Error
}

func getInitData(admin *AdminAccount) initDataSet {
	adminUsername := "admin"
	adminPassword := "3183b16fe3f52e0a2de35b1c665a222a"
	if admin != nil {
		adminUsername = admin.Username
		adminPassword = toolkit.GetPassword(admin.Password, admin.Username)
	}
	resources := currentMenuResources()
	return initDataSet{
		Users: []sysModel.User{
			{
				MODEL:    model(adminUserID, 0, "", seedCreatedAt, "", seedCreatedAt),
				Username: adminUsername,
				Password: adminPassword,
				Sex:      1,
				Status:   1,
			},
		},
		Roles: []sysModel.Role{
			{
				MODEL:  model(adminRoleID, 0, "", seedCreatedAt, "", seedCreatedAt),
				Name:   "系统管理员",
				Alias:  "systemAdmin",
				Status: 1,
				Sort:   1,
			},
		},
		Resources:    resources,
		ResourceApis: []sysModel.ResourceApi{},
		UserRoles: []sysModel.UserRole{
			{ID: 2068217399865245703, UserId: adminUserID, RoleId: adminRoleID},
		},
		RoleResources: buildAdminRoleResources(resources),
	}
}

func currentMenuResources() []sysModel.Resource {
	return []sysModel.Resource{
		menuResource(dashboardResourceID, rootResourceID, "仪表盘", "Dashboard", "/dashboard", "lucide:layout-dashboard", "views/Dashboard.vue", 300, "", true),
		menuResource(aiGatewayResourceID, rootResourceID, "AI 网关", "AiGateway", "/ai-gateway", "lucide:bot", "layout.base", 200, "/ai-gateway/overview", false),
		menuResource(aiOverviewResourceID, aiGatewayResourceID, "数据概览", "AiGatewayOverview", "/ai-gateway/overview", "lucide:bot", "views/ai-gateway/overview/Overview.vue", 700, "", false),
		menuResource(aiProvidersResourceID, aiGatewayResourceID, "LLM 供应商", "AiGatewayProviders", "/ai-gateway/providers", "lucide:network", "views/ai-gateway/providers/ProviderManagement.vue", 600, "", false),
		menuResource(aiChannelsResourceID, aiGatewayResourceID, "渠道号池", "AiGatewayChannels", "/ai-gateway/channels", "lucide:route", "views/ai-gateway/channels/ChannelManagement.vue", 500, "", false),
		menuResource(aiAccountsResourceID, aiGatewayResourceID, "账户管理", "AiGatewayChannelAccounts", "/ai-gateway/channel-accounts", "lucide:users", "views/ai-gateway/channel-accounts/ChannelAccountManagement.vue", 400, "", false),
		menuResource(aiTokensResourceID, aiGatewayResourceID, "API 密钥", "AiGatewayTokens", "/ai-gateway/tokens", "lucide:key-round", "views/ai-gateway/tokens/AccessTokenManagement.vue", 300, "", false),
		menuResource(aiProxiesResourceID, aiGatewayResourceID, "IP 代理池", "AiGatewayProxies", "/ai-gateway/proxies", "lucide:cable", "views/ai-gateway/proxies/ProxyManagement.vue", 200, "", false),
		menuResource(aiLogsResourceID, aiGatewayResourceID, "调用日志", "AiGatewayLogs", "/ai-gateway/logs", "lucide:activity", "views/ai-gateway/calllogs/CallLogsManagement.vue", 100, "", false),
		menuResource(aiApplicationsResourceID, rootResourceID, "AI 应用", "AiApplications", "/ai-applications", "lucide:server", "layout.base", 150, "/ai-applications/code-review", false),
		menuResource(codeReviewResourceID, aiApplicationsResourceID, "AI 代码评审", "CodeReview", "/ai-applications/code-review", "lucide:git-pull-request", "layout.base", 100, "/ai-applications/code-review/projects", false),
		menuResource(remoteAgentResourceID, aiApplicationsResourceID, "Remote Agent", "RemoteAgent", "/ai-applications/remote-agent", "lucide:monitor-cog", "layout.base", 90, "/ai-applications/remote-agent/agents", false),
		menuResource(settingsResourceID, rootResourceID, "系统设置", "Settings", "/settings", "lucide:settings", "layout.base", 100, "/settings/users", false),
		menuResource(settingsUsersResourceID, settingsResourceID, "用户管理", "SettingsUsers", "/settings/users", "lucide:users", "views/settings/users/UserManagement.vue", 500, "", true),
		menuResource(settingsRolesResourceID, settingsResourceID, "角色管理", "SettingsRoles", "/settings/roles", "lucide:shield", "views/settings/roles/RoleManagement.vue", 400, "", true),
		menuResource(settingsMenusResourceID, settingsResourceID, "菜单管理", "SettingsMenus", "/settings/menus", "lucide:list-tree", "views/settings/menus/MenuManagement.vue", 300, "", true),
		menuResource(settingsAccountResourceID, settingsResourceID, "账户设置", "SettingsAccount", "/settings/account", "lucide:user-cog", "views/settings/account/AccountSettings.vue", 200, "", true),
		menuResource(settingsMessageResourceID, settingsResourceID, "我的消息", "MyMessages", "/settings/messages", "lucide:bell", "views/settings/messages/MyMessages.vue", 100, "", true),
	}
}

func buildAdminRoleResources(resources []sysModel.Resource) []sysModel.RoleResource {
	result := make([]sysModel.RoleResource, 0, len(resources))
	for index, item := range resources {
		result = append(result, sysModel.RoleResource{
			ID:         snowflake.ID(roleResourceIDBase + int64(index) + 1),
			RoleId:     adminRoleID,
			ResourceId: item.ID,
		})
	}
	return result
}

func model(id, createdID snowflake.ID, createdBy string, createdAt int64, updatedBy string, updatedAt int64) api.MODEL {
	return api.MODEL{
		ID:        id,
		CreatedId: createdID,
		CreatedBy: createdBy,
		CreatedAt: createdAt,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}
}

func menuResource(id, pid snowflake.ID, title, alias, path, icon, component string, sort uint, redirect string, keepAlive bool) sysModel.Resource {
	return sysModel.Resource{
		MODEL:       model(id, adminUserID, "admin", seedCreatedAt, "admin", seedCreatedAt),
		Pid:         pid,
		Title:       title,
		Alias:       alias,
		Type:        1,
		Redirect:    redirect,
		Path:        path,
		Icon:        icon,
		Status:      1,
		Sort:        sort,
		Component:   component,
		Hidden:      false,
		ParentRoute: "",
		KeepAlive:   keepAlive,
		Query:       "{}",
	}
}

func validateCurrentMenuResources(resources []sysModel.Resource) error {
	seenIDs := make(map[snowflake.ID]struct{}, len(resources))
	seenPaths := make(map[string]struct{}, len(resources))
	for _, item := range resources {
		if item.ID == 0 || item.Path == "" || item.Title == "" || item.Alias == "" {
			return fmt.Errorf("invalid seed menu: %q", item.Title)
		}
		if _, exists := seenIDs[item.ID]; exists {
			return fmt.Errorf("duplicate seed menu id: %s", item.ID.String())
		}
		if _, exists := seenPaths[item.Path]; exists {
			return fmt.Errorf("duplicate seed menu path: %s", item.Path)
		}
		seenIDs[item.ID] = struct{}{}
		seenPaths[item.Path] = struct{}{}
	}
	return nil
}
