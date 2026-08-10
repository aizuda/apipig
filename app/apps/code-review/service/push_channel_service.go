package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	aiService "apipig/app/ai/service"
	reviewModel "apipig/app/apps/code-review/model"
	reviewReq "apipig/app/apps/code-review/model/request"
	reviewResp "apipig/app/apps/code-review/model/response"
	wechatService "apipig/app/apps/wechat-bot/service"
	"apipig/core/api"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

type pushChannelService struct{ vault aiService.CredentialVault }

// PushChannelFacade is the admin API boundary for channel configuration.
type PushChannelFacade = pushChannelService

func (s *pushChannelService) List(projectID snowflake.ID) ([]reviewModel.PushChannelView, error) {
	return s.list(projectID, false)
}

// ListForEdit returns decrypted channel configuration for the authenticated
// admin editor. General project responses continue to use List and stay masked.
func (s *pushChannelService) ListForEdit(projectID snowflake.ID) ([]reviewModel.PushChannelView, error) {
	return s.list(projectID, true)
}

func (s *pushChannelService) list(projectID snowflake.ID, revealSecrets bool) ([]reviewModel.PushChannelView, error) {
	var rows []reviewModel.PushChannel
	if err := global.DB.Where("project_id = ?", projectID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]reviewModel.PushChannelView, 0, len(rows))
	for _, row := range rows {
		view, err := s.toView(row, revealSecrets)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *pushChannelService) Save(params *reviewReq.PushChannelsSaveRequest) (reviewResp.PushChannelsResult, error) {
	var result reviewResp.PushChannelsResult
	if params == nil || params.ProjectID == 0 {
		return result, errors.New("项目 ID 不能为空")
	}
	var project reviewModel.Project
	if err := global.DB.First(&project, params.ProjectID).Error; err != nil {
		return result, err
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var existingRows []reviewModel.PushChannel
		if err := tx.Where("project_id = ?", params.ProjectID).Find(&existingRows).Error; err != nil {
			return err
		}
		existingByID := make(map[snowflake.ID]string, len(existingRows))
		for _, row := range existingRows {
			existingByID[row.ID] = row.Config
		}
		if err := tx.Where("project_id = ?", params.ProjectID).Delete(&reviewModel.PushChannel{}).Error; err != nil {
			return err
		}
		for _, input := range params.Channels {
			row, err := s.fromRequest(params.ProjectID, input, existingByID[input.ID])
			if err != nil {
				return err
			}
			row.MODEL = apiModelForChannel()
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Channels, err = s.List(params.ProjectID)
	return result, err
}

func apiModelForChannel() api.MODEL {
	return api.MODEL{ID: db.GetId(), CreatedAt: time.Now().UnixMilli()}
}

func (s *pushChannelService) Test(params *reviewReq.PushChannelTestRequest) (bool, error) {
	if params == nil || params.ProjectID == 0 {
		return false, errors.New("项目 ID 不能为空")
	}
	var existing string
	if params.Channel.ID != 0 {
		var saved reviewModel.PushChannel
		if err := global.DB.Where("id = ? AND project_id = ?", params.Channel.ID, params.ProjectID).First(&saved).Error; err == nil {
			existing = saved.Config
		}
	}
	row, err := s.fromRequest(params.ProjectID, params.Channel, existing)
	if err != nil {
		return false, err
	}
	var project reviewModel.Project
	if err = global.DB.First(&project, params.ProjectID).Error; err != nil {
		return false, err
	}
	config, err := s.decryptConfig(row.Config)
	if err != nil {
		return false, err
	}
	return s.send(context.Background(), params.Channel.Type, config, "代码评审渠道测试\n项目："+project.Name+"\n消息发送正常")
}

func (s *pushChannelService) fromRequest(projectID snowflake.ID, input reviewReq.PushChannelRequest, existing string) (reviewModel.PushChannel, error) {
	typ := strings.TrimSpace(strings.ToLower(input.Type))
	if typ != reviewModel.PushChannelTypeWeCom && typ != reviewModel.PushChannelTypeDingTalk && typ != reviewModel.PushChannelTypeWechatBot {
		return reviewModel.PushChannel{}, errors.New("不支持的推送渠道类型")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = channelTypeName(typ)
	}
	config := make(map[string]string, len(input.Config))
	for k, v := range input.Config {
		config[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if existing != "" {
		old, _ := s.decryptConfig(existing)
		for key, value := range old {
			if config[key] == "" {
				config[key] = value
			}
		}
	}
	if err := validateChannelConfig(typ, config); err != nil {
		return reviewModel.PushChannel{}, err
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return reviewModel.PushChannel{}, err
	}
	enc, err := s.vault.Encrypt(string(raw))
	if err != nil {
		return reviewModel.PushChannel{}, err
	}
	return reviewModel.PushChannel{MODEL: reviewModel.PushChannel{}.MODEL, ProjectID: projectID, Type: typ, Name: name, Enabled: input.Enabled, Config: enc}, nil
}

func (s *pushChannelService) toView(row reviewModel.PushChannel, revealSecrets bool) (reviewModel.PushChannelView, error) {
	config, err := s.decryptConfig(row.Config)
	if err != nil {
		return reviewModel.PushChannelView{}, err
	}
	secretConfigured := false
	for key, value := range config {
		if strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "token") || key == "webhookUrl" {
			secretConfigured = secretConfigured || value != ""
			if !revealSecrets {
				config[key] = ""
			}
		}
	}
	return reviewModel.PushChannelView{ID: row.ID, ProjectID: row.ProjectID, Type: row.Type, Name: row.Name, Enabled: row.Enabled, Config: config, SecretConfigured: secretConfigured}, nil
}

func (s *pushChannelService) decryptConfig(raw string) (map[string]string, error) {
	plain, err := s.vault.Decrypt(raw)
	if err != nil {
		return nil, err
	}
	var config map[string]string
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return nil, err
	}
	return config, nil
}

func validateChannelConfig(typ string, config map[string]string) error {
	switch typ {
	case reviewModel.PushChannelTypeWeCom, reviewModel.PushChannelTypeDingTalk:
		if strings.TrimSpace(config["webhookUrl"]) == "" {
			return errors.New("请填写机器人 Webhook 地址")
		}
		parsed, err := url.ParseRequestURI(config["webhookUrl"])
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return errors.New("Webhook 地址格式不正确")
		}
	case reviewModel.PushChannelTypeWechatBot:
		if config["botId"] == "" {
			return errors.New("请选择微信 Bot")
		}
	}
	return nil
}

func channelTypeName(typ string) string {
	switch typ {
	case reviewModel.PushChannelTypeWeCom:
		return "企业微信机器人"
	case reviewModel.PushChannelTypeDingTalk:
		return "钉钉机器人"
	default:
		return "微信 Bot"
	}
}

func (s *pushChannelService) push(task reviewModel.Task, project reviewModel.Project) {
	var rows []reviewModel.PushChannel
	if err := global.DB.Where("project_id = ? AND enabled = ?", project.ID, true).Find(&rows).Error; err != nil {
		logReviewError("加载推送渠道失败", err)
		return
	}
	message := buildReviewPushMessage(task, project)
	for _, row := range rows {
		config, err := s.decryptConfig(row.Config)
		if err != nil {
			logReviewError("解密推送渠道失败", err)
			continue
		}
		if _, err = s.send(context.Background(), row.Type, config, message); err != nil {
			logReviewError("发送代码评审结果失败", err)
		}
	}
}

const reviewPushFindingLimit = 3

// buildReviewPushMessage intentionally leaves the full Markdown report in the
// system. Push channels only need a compact summary that can be scanned quickly.
func buildReviewPushMessage(task reviewModel.Task, project reviewModel.Project) string {
	summary := strings.TrimSpace(task.Summary)
	if summary == "" {
		summary = "暂无风险摘要"
	}
	summary = truncateReviewPushText(summary, 300)

	var findings []reviewResp.Finding
	if strings.TrimSpace(task.Findings) != "" {
		_ = json.Unmarshal([]byte(task.Findings), &findings)
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "代码评审结果\n项目：%s\n仓库：%s\n标题：%s\n风险等级：%s\n风险摘要：%s\n\n重点提示：", project.Name, task.RepositoryName, task.Title, task.RiskLevel, summary)
	if len(findings) == 0 {
		builder.WriteString("\n暂无结构化问题")
	} else {
		limit := len(findings)
		if limit > reviewPushFindingLimit {
			limit = reviewPushFindingLimit
		}
		for _, finding := range findings[:limit] {
			builder.WriteString("\n- ")
			builder.WriteString(formatReviewPushFinding(finding))
		}
		if remaining := len(findings) - limit; remaining > 0 {
			fmt.Fprintf(&builder, "\n还有 %d 条问题，请前往系统查看完整报告", remaining)
		}
	}
	builder.WriteString("\n\n更多详情请前往系统“AI 应用 > 代码评审 > 评审任务”查看完整报告。")
	return builder.String()
}

func formatReviewPushFinding(finding reviewResp.Finding) string {
	severity := strings.TrimSpace(finding.Severity)
	if severity == "" {
		severity = "提示"
	}
	title := strings.TrimSpace(finding.Title)
	if title == "" {
		title = strings.TrimSpace(finding.Description)
	}
	if title == "" {
		title = "发现待关注问题"
	}
	title = truncateReviewPushText(title, 160)
	location := strings.TrimSpace(finding.File)
	if location != "" && finding.Line > 0 {
		location = fmt.Sprintf("%s:%d", location, finding.Line)
	}
	if location != "" {
		return fmt.Sprintf("[%s] %s（%s）", severity, title, location)
	}
	return fmt.Sprintf("[%s] %s", severity, title)
}

func truncateReviewPushText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 || len([]rune(value)) <= maxRunes {
		return value
	}
	return string([]rune(value)[:maxRunes-1]) + "…"
}

func (s *pushChannelService) send(ctx context.Context, typ string, config map[string]string, message string) (bool, error) {
	if typ == reviewModel.PushChannelTypeWechatBot {
		botID, err := snowflake.ParseString(config["botId"])
		if err != nil {
			return false, errors.New("微信 Bot ID 不正确")
		}
		return true, wechatService.WechatBotService.BotService.SendTakeover(ctx, botID, config["userId"], message)
	}
	payload := map[string]any{"msgtype": "text", "text": map[string]string{"content": message}}
	if typ == reviewModel.PushChannelTypeDingTalk {
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": message}}
	}
	body, _ := json.Marshal(payload)
	target := config["webhookUrl"]
	if typ == reviewModel.PushChannelTypeDingTalk && strings.TrimSpace(config["secret"]) != "" {
		// DingTalk robot signing: timestamp + HMAC-SHA256 secret, URL encoded.
		timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
		sign := hmac.New(sha256.New, []byte(config["secret"]))
		_, _ = sign.Write([]byte(timestamp + "\n" + config["secret"]))
		separator := "?"
		if strings.Contains(target, "?") {
			separator = "&"
		}
		target += separator + "timestamp=" + url.QueryEscape(timestamp) + "&sign=" + url.QueryEscape(base64.StdEncoding.EncodeToString(sign.Sum(nil)))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("推送返回 HTTP %d", resp.StatusCode)
	}
	return true, nil
}
