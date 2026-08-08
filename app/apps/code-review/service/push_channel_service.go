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
	var rows []reviewModel.PushChannel
	if err := global.DB.Where("project_id = ?", projectID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]reviewModel.PushChannelView, 0, len(rows))
	for _, row := range rows {
		view, err := s.toView(row)
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

func (s *pushChannelService) toView(row reviewModel.PushChannel) (reviewModel.PushChannelView, error) {
	config, err := s.decryptConfig(row.Config)
	if err != nil {
		return reviewModel.PushChannelView{}, err
	}
	secretConfigured := false
	for key, value := range config {
		if strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "token") || key == "webhookUrl" {
			secretConfigured = secretConfigured || value != ""
			config[key] = ""
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
		if config["botId"] == "" || config["userId"] == "" {
			return errors.New("请选择微信 Bot 并填写接收人 ID")
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
	message := fmt.Sprintf("代码评审结果\n项目：%s\n仓库：%s\n标题：%s\n风险：%s\n\n%s", project.Name, task.RepositoryName, task.Title, task.RiskLevel, task.Summary)
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
