package service

import (
	"context"
	"time"

	"apipig/global"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AiServiceGroup struct {
	GatewayService        *GatewayService
	ProviderService       *ProviderService
	ChannelService        *ChannelService
	ChannelAccountService *ChannelAccountService
	AccessTokenService    *AccessTokenService
	AccessTokenTagService *AccessTokenTagService
	ProxyService          *ProxyService
	CallLogService        *CallLogService
	Vault                 CredentialVault
}

func NewAIServiceGroup(vault CredentialVault) *AiServiceGroup {
	if vault == nil {
		vault = newAESCredentialVault(func() string { return global.CONFIG.AI.EncryptionKey })
	}
	store := newGormAIStore(func() *gorm.DB { return global.DB })
	repository := newGormGatewayRepository(func() *gorm.DB { return global.DB })
	logSink := newAsyncCallLogSink(repository, func() AsyncLogOptions {
		return AsyncLogOptions{
			QueueSize:     global.CONFIG.AI.LogQueueSize,
			BatchSize:     global.CONFIG.AI.LogBatchSize,
			FlushInterval: time.Duration(global.CONFIG.AI.LogFlushIntervalMs) * time.Millisecond,
		}
	}, func(err error) {
		if global.LOG != nil {
			global.LOG.Error("AI 调用日志异步写入失败", zap.Error(err))
		}
	})
	gateway := NewGatewayService(GatewayDependencies{Vault: vault, Repository: repository, LogSink: logSink})
	group := &AiServiceGroup{
		GatewayService:        gateway,
		ProviderService:       &ProviderService{gateway: gateway, store: store},
		ChannelService:        &ChannelService{gateway: gateway, store: store},
		ChannelAccountService: &ChannelAccountService{gateway: gateway, vault: vault, store: store},
		AccessTokenService:    &AccessTokenService{store: store},
		AccessTokenTagService: &AccessTokenTagService{store: store},
		ProxyService:          &ProxyService{gateway: gateway, vault: vault, store: store},
		CallLogService:        &CallLogService{store: store},
		Vault:                 vault,
	}
	return group
}

var AiService = NewAIServiceGroup(nil)

func ShutdownAI(ctx context.Context) error {
	if AiService == nil || AiService.GatewayService == nil || AiService.GatewayService.logSink == nil {
		return nil
	}
	return AiService.GatewayService.logSink.Close(ctx)
}
