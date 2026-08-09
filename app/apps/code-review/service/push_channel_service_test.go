package service

import (
	"strings"
	"testing"

	reviewModel "apipig/app/apps/code-review/model"
	"apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pushChannelTestVault struct{}

func (pushChannelTestVault) Encrypt(value string) (string, error) {
	return "encrypted:" + value, nil
}

func (pushChannelTestVault) Decrypt(value string) (string, error) {
	return strings.TrimPrefix(value, "encrypted:"), nil
}

func (pushChannelTestVault) Enabled() bool { return true }

func TestPushChannelViewSecretVisibility(t *testing.T) {
	service := &pushChannelService{vault: pushChannelTestVault{}}
	row := reviewModel.PushChannel{
		MODEL:     api.MODEL{ID: snowflake.ID(101)},
		ProjectID: snowflake.ID(201),
		Type:      reviewModel.PushChannelTypeDingTalk,
		Name:      "DingTalk",
		Enabled:   true,
		Config:    `encrypted:{"webhookUrl":"https://example.test/hook/token","secret":"signing-secret"}`,
	}

	masked, err := service.toView(row, false)
	require.NoError(t, err)
	assert.Empty(t, masked.Config["webhookUrl"])
	assert.Empty(t, masked.Config["secret"])
	assert.True(t, masked.SecretConfigured)

	revealed, err := service.toView(row, true)
	require.NoError(t, err)
	assert.Equal(t, "https://example.test/hook/token", revealed.Config["webhookUrl"])
	assert.Equal(t, "signing-secret", revealed.Config["secret"])
	assert.True(t, revealed.SecretConfigured)
}

func TestBuildReviewPushMessageIsCompact(t *testing.T) {
	task := reviewModel.Task{
		RepositoryName: "acme/api",
		Title:          "修复鉴权",
		RiskLevel:      "high",
		Summary:        "存在需要优先处理的安全风险",
		Findings: `[{
			"severity":"high","file":"api/auth.go","line":42,"title":"接口缺少鉴权",
			"description":"不应允许未登录请求通过"
		},{"severity":"medium","file":"api/user.go","line":18,"title":"错误信息可能泄露内部细节"},
		{"severity":"low","file":"README.md","line":3,"title":"补充测试说明"},
		{"severity":"medium","file":"api/log.go","line":9,"title":"日志字段缺失"}]`,
	}
	message := buildReviewPushMessage(task, reviewModel.Project{Name: "API 服务"})

	assert.Contains(t, message, "风险等级：high")
	assert.Contains(t, message, "风险摘要：存在需要优先处理的安全风险")
	assert.Contains(t, message, "重点提示：")
	assert.Contains(t, message, "[high] 接口缺少鉴权（api/auth.go:42）")
	assert.Contains(t, message, "还有 1 条问题，请前往系统查看完整报告")
	assert.Contains(t, message, "更多详情请前往系统“AI 应用 > 代码评审 > 评审任务”查看完整报告。")
	assert.NotContains(t, message, "不应允许未登录请求通过")
}

func TestBuildReviewPushMessageHandlesMissingSummaryAndFindings(t *testing.T) {
	message := buildReviewPushMessage(reviewModel.Task{}, reviewModel.Project{})

	assert.Contains(t, message, "风险摘要：暂无风险摘要")
	assert.Contains(t, message, "重点提示：\n暂无结构化问题")
	assert.Contains(t, message, "前往系统")
}
