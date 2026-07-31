package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAIReview(t *testing.T) {
	result := parseAIReview("```json\n{\"riskLevel\":\"high\",\"summary\":\"存在权限绕过\",\"findings\":[{\"severity\":\"high\",\"file\":\"api.go\",\"line\":42,\"category\":\"security\",\"title\":\"缺少鉴权\",\"description\":\"接口未校验用户\",\"suggestion\":\"增加鉴权中间件\"}],\"report\":\"# 评审\"}\n```")
	require.Equal(t, "high", result.RiskLevel)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "api.go", result.Findings[0].File)
}

func TestParseAIReviewFallback(t *testing.T) {
	result := parseAIReview("普通文本报告")
	require.Equal(t, "unknown", result.RiskLevel)
	require.Equal(t, "普通文本报告", result.Report)
}
