package initialize

import (
	"testing"

	remoteAgentModel "apipig/app/apps/remote-agent/model"
	"apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type legacyConversation struct {
	api.MODEL
	AgentID          snowflake.ID `gorm:"type:bigint;not null;index"`
	Title            string       `gorm:"size:200;not null;index"`
	WorkingDirectory string       `gorm:"size:500;not null"`
	Status           string       `gorm:"size:20;not null;index"`
	Pinned           bool         `gorm:"not null;default:false;index"`
	PinnedAt         int64        `gorm:"type:bigint;not null;default:0;index"`
	LastMessageAt    int64        `gorm:"type:bigint;not null;index"`
	PermissionMode   string       `gorm:"size:20;not null;default:FULL_ACCESS"`
	ControlMode      string       `gorm:"size:20;not null;default:WEB;index"`
	WechatBotID      snowflake.ID `gorm:"type:bigint;not null;default:0;index"`
	WechatUserID     string       `gorm:"size:200;not null;default:''"`
	WechatTakeoverAt int64        `gorm:"type:bigint;not null;default:0"`
}

func (legacyConversation) TableName() string { return "ap_remote_agent_conversation" }

func TestConversationMigrationAddsCLITypeToPopulatedLegacyTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:conversation-migration?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 模拟旧版本没有 cli_type 的会话表，并保留一条历史数据。
	require.NoError(t, db.AutoMigrate(&legacyConversation{}))
	require.NoError(t, db.Create(&legacyConversation{
		MODEL: coreAPIModel(1), AgentID: 2, Title: "legacy", WorkingDirectory: ".", Status: "ACTIVE", LastMessageAt: 1,
	}).Error)

	require.NoError(t, db.AutoMigrate(&remoteAgentModel.Conversation{}))

	var conversation remoteAgentModel.Conversation
	require.NoError(t, db.First(&conversation, 1).Error)
	require.Equal(t, remoteAgentModel.CLITypeCodex, conversation.CLIType)
}

func coreAPIModel(id int64) api.MODEL {
	return api.MODEL{ID: snowflake.ID(id), CreatedAt: 1, UpdatedAt: 1}
}
