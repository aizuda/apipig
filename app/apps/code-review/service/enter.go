package service

import (
	"context"
	"sync"

	aiService "apipig/app/ai/service"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"go.uber.org/zap"
)

type ReviewServiceGroup struct {
	ProjectService     *ProjectService
	TaskService        *TaskService
	WebhookService     *WebhookService
	PushChannelService *pushChannelService
}

type taskRunner struct {
	queue         chan snowflake.ID
	stop          chan struct{}
	done          chan struct{}
	startOnce     sync.Once
	stopOnce      sync.Once
	gateway       *aiService.GatewayService
	startExecutor func(snowflake.ID)
}

func NewReviewServiceGroup() *ReviewServiceGroup {
	runner := &taskRunner{gateway: aiService.AiService.GatewayService}
	pushService := &pushChannelService{vault: aiService.AiService.Vault}
	projectService := &ProjectService{vault: aiService.AiService.Vault, push: pushService}
	taskService := &TaskService{runner: runner}
	runnerService := &reviewExecutor{runner: runner, vault: aiService.AiService.Vault, push: pushService}
	runner.startExecutor = runnerService.execute
	return &ReviewServiceGroup{
		ProjectService:     projectService,
		TaskService:        taskService,
		WebhookService:     &WebhookService{vault: aiService.AiService.Vault, runner: runner},
		PushChannelService: pushService,
	}
}

var ReviewService = NewReviewServiceGroup()

func StartReview() {
	if ReviewService != nil && ReviewService.TaskService != nil {
		ReviewService.TaskService.Start()
	}
}

func ShutdownReview(ctx context.Context) error {
	if ReviewService == nil || ReviewService.TaskService == nil {
		return nil
	}
	return ReviewService.TaskService.Shutdown(ctx)
}

func logReviewError(message string, err error) {
	if global.LOG != nil {
		global.LOG.Error(message, zap.Error(err))
	}
}
