package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiService "apipig/app/ai/service"
	reviewModel "apipig/app/apps/code-review/model"
	reviewResp "apipig/app/apps/code-review/model/response"
	"apipig/global"
	"apipig/toolkit"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

type reviewExecutor struct {
	runner *taskRunner
	vault  aiService.CredentialVault
}

func (e *reviewExecutor) execute(id snowflake.ID) {
	result := global.DB.Model(&reviewModel.Task{}).Where("id = ? AND status = ?", id, reviewModel.TaskStatusQueued).Updates(map[string]any{"status": reviewModel.TaskStatusRunning, "started_at": toolkit.GetNowUnixMilli(), "attempt": gorm.Expr("attempt + 1")})
	if result.Error != nil || result.RowsAffected == 0 {
		return
	}
	var task reviewModel.Task
	if err := global.DB.First(&task, id).Error; err != nil {
		return
	}
	var project reviewModel.Project
	if err := global.DB.First(&project, task.ProjectID).Error; err != nil {
		e.fail(id, err)
		return
	}
	if project.Status != reviewModel.ProjectStatusEnabled {
		e.ignore(id, "项目已禁用")
		return
	}
	repositoryToken, err := e.vault.Decrypt(project.RepositoryToken)
	if err != nil {
		e.fail(id, err)
		return
	}
	gitTimeout := global.CONFIG.CodeReview.GitTimeoutSec
	if gitTimeout <= 0 {
		gitTimeout = 120
	}
	gitCtx, cancelGit := context.WithTimeout(context.Background(), time.Duration(gitTimeout)*time.Second)
	diff, err := prepareDiff(gitCtx, project, task, repositoryToken, defaultMaxDiffBytes(), defaultMaxChangedFiles())
	cancelGit()
	if err != nil {
		e.fail(id, err)
		return
	}
	if strings.TrimSpace(diff.Content) == "" {
		_ = global.DB.Model(&task).Updates(map[string]any{"status": reviewModel.TaskStatusIgnored, "summary": "本次变更没有可评审的文本差异", "changed_files": diff.ChangedFiles, "additions": diff.Additions, "deletions": diff.Deletions, "finished_at": toolkit.GetNowUnixMilli()}).Error
		return
	}
	aiTimeout := global.CONFIG.CodeReview.AITimeoutSec
	if aiTimeout <= 0 {
		aiTimeout = 180
	}
	aiCtx, cancelAI := context.WithTimeout(context.Background(), time.Duration(aiTimeout)*time.Second)
	raw, err := e.runner.gateway.GenerateInternal(aiService.InternalGenerateParams{Context: aiCtx, AccessTokenID: project.AccessTokenID, Model: project.Model, SystemPrompt: reviewSystemPrompt(project.ReviewPrompt), UserPrompt: reviewUserPrompt(task, diff), MaxTokens: 4096, Path: "/internal/apps/code-review"})
	cancelAI()
	if err != nil {
		e.fail(id, err)
		return
	}
	review := parseAIReview(raw)
	findings, _ := json.Marshal(review.Findings)
	if err = global.DB.Model(&task).Updates(map[string]any{
		"status": reviewModel.TaskStatusSucceeded, "changed_files": diff.ChangedFiles, "additions": diff.Additions,
		"deletions": diff.Deletions, "diff_truncated": diff.Truncated, "risk_level": review.RiskLevel,
		"summary": review.Summary, "report": review.Report, "findings": string(findings), "finished_at": toolkit.GetNowUnixMilli(), "error_message": "",
	}).Error; err != nil {
		logReviewError("保存代码评审报告失败", err)
	}
}

func (e *reviewExecutor) fail(id snowflake.ID, err error) {
	message := err.Error()
	if len(message) > 2000 {
		message = message[:2000]
	}
	_ = global.DB.Model(&reviewModel.Task{}).Where("id = ?", id).Updates(map[string]any{"status": reviewModel.TaskStatusFailed, "error_message": message, "finished_at": toolkit.GetNowUnixMilli()}).Error
}
func (e *reviewExecutor) ignore(id snowflake.ID, reason string) {
	_ = global.DB.Model(&reviewModel.Task{}).Where("id = ?", id).Updates(map[string]any{"status": reviewModel.TaskStatusIgnored, "error_message": reason, "finished_at": toolkit.GetNowUnixMilli()}).Error
}

func reviewSystemPrompt(custom string) string {
	base := `你是资深软件工程师和安全审计专家。只评审给出的 Git diff，不推测未展示的代码。重点检查正确性、安全、并发、数据一致性、性能、可维护性和测试缺失。忽略纯格式偏好。必须返回 JSON 对象，字段为 riskLevel(low|medium|high|critical)、summary、findings、report。findings 每项字段为 severity、file、line、category、title、description、suggestion。若无问题 findings 返回空数组。report 使用中文 Markdown。`
	if strings.TrimSpace(custom) != "" {
		base += "\n项目补充规则：\n" + strings.TrimSpace(custom)
	}
	return base
}

func reviewUserPrompt(task reviewModel.Task, diff diffResult) string {
	truncated := "否"
	if diff.Truncated {
		truncated = "是，结论中必须明确说明评审范围受限"
	}
	return fmt.Sprintf("仓库：%s\n事件：%s\n标题：%s\n作者：%s\nBase SHA：%s\nHead SHA：%s\n文件数：%d\n新增：%d\n删除：%d\nDiff 是否截断：%s\n\n```diff\n%s\n```", task.RepositoryName, task.EventType, task.Title, task.Author, task.BaseSHA, task.HeadSHA, diff.ChangedFiles, diff.Additions, diff.Deletions, truncated, diff.Content)
}

func parseAIReview(raw string) reviewResp.AIReviewResult {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)
	var result reviewResp.AIReviewResult
	if err := json.Unmarshal([]byte(clean), &result); err != nil {
		return reviewResp.AIReviewResult{RiskLevel: "unknown", Summary: "AI 已完成评审，但返回内容不是标准 JSON", Report: raw, Findings: []reviewResp.Finding{}}
	}
	if result.RiskLevel == "" {
		result.RiskLevel = "low"
	}
	if result.Report == "" {
		result.Report = result.Summary
	}
	if result.Findings == nil {
		result.Findings = []reviewResp.Finding{}
	}
	return result
}

func defaultMaxDiffBytes() int {
	value := global.CONFIG.CodeReview.MaxDiffBytes
	if value <= 0 {
		return 512 * 1024
	}
	return value
}
func defaultMaxChangedFiles() int {
	value := global.CONFIG.CodeReview.MaxChangedFiles
	if value <= 0 {
		return 200
	}
	return value
}
