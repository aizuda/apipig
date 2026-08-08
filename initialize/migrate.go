package initialize

import (
	aiModel "apipig/app/ai/model"
	reviewModel "apipig/app/apps/code-review/model"
	remoteAgentModel "apipig/app/apps/remote-agent/model"
	wechatBotModel "apipig/app/apps/wechat-bot/model"
	sysModel "apipig/app/sys/model"
	"log"

	"gorm.io/gorm"
)

func autoMigrate(db *gorm.DB) error {
	log.Println("Initiating migration...")
	if err := db.AutoMigrate(migrationModels()...); err != nil {
		return err
	}
	log.Println("Migration Completed...")
	return nil
}

// migrationModels is the single source of truth for tables managed by GORM.
// Keeping the list reusable lets compatibility tests cover every migrated model.
func migrationModels() []any {
	return []any{
		&aiModel.Provider{},
		&aiModel.Channel{},
		&aiModel.ChannelAccount{},
		&aiModel.AccessToken{},
		&aiModel.AccessTokenTag{},
		&aiModel.AccessTokenTagRelation{},
		&aiModel.Proxy{},
		&aiModel.CallLog{},
		&reviewModel.Project{},
		&reviewModel.Task{},
		&reviewModel.PushChannel{},
		&remoteAgentModel.Agent{},
		&remoteAgentModel.Heartbeat{},
		&remoteAgentModel.Conversation{},
		&remoteAgentModel.Message{},
		&remoteAgentModel.MessageChunk{},
		&remoteAgentModel.Command{},
		&wechatBotModel.Bot{},
		&wechatBotModel.Contact{},
		&wechatBotModel.Message{},
		&sysModel.Resource{},
		&sysModel.ResourceApi{},
		&sysModel.Role{},
		&sysModel.RoleResource{},
		&sysModel.User{},
		&sysModel.UserRole{},
		&sysModel.UserSession{},
	}
}
