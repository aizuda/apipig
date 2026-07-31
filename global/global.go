package global

import (
	"apipig/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	LOG    *zap.Logger
	CONFIG config.Server
)

var (

	// 规则引擎相关变量
	RuleInit     = false
	RuleProperty = false
	RuleEvent    = false
	RuleStatus   = false

	// 场景监听相关变量
	SceneInit     = false
	SceneProperty = false
	SceneEvent    = false
	SceneStatus   = false
)
