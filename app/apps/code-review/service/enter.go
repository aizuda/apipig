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
	ProjectService *ProjectService
	TaskService    *TaskService
	WebhookService *WebhookService
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
	projectService := &ProjectService{vault: aiService.AiService.Vault}
	taskService := &TaskService{runner: runner}
	runnerService := &reviewExecutor{runner: runner, vault: aiService.AiService.Vault}
	runner.startExecutor = runnerService.execute
	return &ReviewServiceGroup{
		ProjectService: projectService,
		TaskService:    taskService,
		WebhookService: &WebhookService{vault: aiService.AiService.Vault, runner: runner},
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
