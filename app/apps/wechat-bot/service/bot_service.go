package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	aiService "apipig/app/ai/service"
	wechatModel "apipig/app/apps/wechat-bot/model"
	wechatReq "apipig/app/apps/wechat-bot/model/request"
	wechatResp "apipig/app/apps/wechat-bot/model/response"
	coreResponse "apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	ilink "github.com/openilink/openilink-sdk-go"
	"gorm.io/gorm"
)

var ErrWebhookUnauthorized = errors.New("Webhook 认证失败")

const webhookTimestampSkew = 5 * time.Minute

type bindSession struct {
	mu          sync.Mutex
	client      *ilink.Client
	qrCode      string
	pollBaseURL string
	name        string
	targetBotID snowflake.ID
	createdBy   wechatModel.Bot
	expiresAt   time.Time
	result      *wechatResp.BindStatusResult
}

type BotService struct {
	vault           aiService.CredentialVault
	runtime         *Runtime
	bindMu          sync.RWMutex
	sessions        map[string]*bindSession
	credentialMu    sync.Mutex
	outboundHandler func(snowflake.ID, string, string) error
}

func (s *BotService) SetInboundHandler(handler func(snowflake.ID, string, string) error) {
	s.runtime.SetInboundHandler(handler)
}

// SetOutboundHandler mirrors messages sent manually from the Bot console into
// an Agent conversation while that contact is under takeover.
func (s *BotService) SetOutboundHandler(handler func(snowflake.ID, string, string) error) {
	s.outboundHandler = handler
}

func (s *BotService) SendTakeover(ctx context.Context, botID snowflake.ID, userID, content string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		var contact wechatModel.Contact
		cutoff := time.Now().Add(-sendWindow).UnixMilli()
		if err := global.DB.Where("bot_record_id = ? AND last_active_at > ?", botID, cutoff).
			Order("last_active_at DESC, id DESC").First(&contact).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("没有可发送的有效会话，请先让用户向 Bot 发送一条消息")
			}
			return err
		}
		userID = contact.UserID
	}
	_, err := s.runtime.Send(ctx, botID, userID, content)
	return err
}

func newBotService(vault aiService.CredentialVault) *BotService {
	service := &BotService{vault: vault, sessions: make(map[string]*bindSession)}
	service.runtime = newRuntime(vault)
	return service
}

func (s *BotService) Page(params *wechatReq.BotPageParams) (coreResponse.PageResult, error) {
	query := global.DB.Model(&wechatModel.Bot{})
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			like := "%" + keyword + "%"
			query = query.Where("name LIKE ? OR bot_id LIKE ? OR ilink_user_id LIKE ?", like, like, like)
		}
		if status := strings.TrimSpace(params.Status); status != "" {
			query = query.Where("status = ?", strings.ToUpper(status))
		}
	}
	var bots []wechatModel.Bot
	return db.Page(query.Order("created_at DESC"), params.GetPageInfo(), bots)
}

func (s *BotService) List(params *wechatReq.BotListParams) ([]wechatModel.Bot, error) {
	query := global.DB.Model(&wechatModel.Bot{})
	if params != nil {
		if name := strings.TrimSpace(params.Name); name != "" {
			query = query.Where("name LIKE ?", "%"+name+"%")
		}
		if status := strings.TrimSpace(params.Status); status != "" {
			query = query.Where("status = ?", strings.ToUpper(status))
		}
		if params.Enabled != nil {
			query = query.Where("enabled = ?", *params.Enabled)
		}
	}
	bots := make([]wechatModel.Bot, 0)
	err := query.Order("name ASC, created_at DESC").Limit(50).Find(&bots).Error
	return bots, err
}

func (s *BotService) StartBind(params *wechatReq.BindStartParams) (wechatResp.BindStartResult, error) {
	var result wechatResp.BindStartResult
	if params == nil || params.Ctx == nil {
		return result, errors.New("扫码绑定参数不能为空")
	}
	name := strings.TrimSpace(params.Request.Name)
	if len([]rune(name)) > 100 {
		return result, errors.New("Bot 名称不能超过 100 个字符")
	}
	if params.Request.BotID != 0 {
		var existing wechatModel.Bot
		if err := global.DB.First(&existing, params.Request.BotID).Error; err != nil {
			return result, err
		}
		if name == "" {
			name = existing.Name
		}
	}
	ctx, cancel := context.WithTimeout(params.Ctx.Context(), 20*time.Second)
	defer cancel()
	client := ilink.NewClient("")
	qr, err := client.FetchQRCode(ctx)
	if err != nil {
		return result, fmt.Errorf("获取微信登录二维码: %w", err)
	}
	sessionID, err := randomSessionID()
	if err != nil {
		return result, err
	}
	seed := wechatModel.Bot{MODEL: db.NewModel(params.Ctx)}
	session := &bindSession{
		client: client, qrCode: qr.QRCode, pollBaseURL: ilink.DefaultBaseURL,
		name: name, targetBotID: params.Request.BotID, createdBy: seed,
		expiresAt: time.Now().Add(10 * time.Minute),
	}
	s.bindMu.Lock()
	s.sessions[sessionID] = session
	s.bindMu.Unlock()
	time.AfterFunc(10*time.Minute, func() { s.removeSession(sessionID, session) })
	return wechatResp.BindStartResult{SessionID: sessionID, QRCode: qr.QRCodeImgContent}, nil
}

func (s *BotService) PollBind(params *wechatReq.BindStatusParams) (wechatResp.BindStatusResult, error) {
	var empty wechatResp.BindStatusResult
	if params == nil || strings.TrimSpace(params.SessionID) == "" {
		return empty, errors.New("扫码会话不能为空")
	}
	s.bindMu.RLock()
	session := s.sessions[params.SessionID]
	s.bindMu.RUnlock()
	if session == nil {
		return empty, errors.New("扫码会话不存在或已过期")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.result != nil {
		return *session.result, nil
	}
	if time.Now().After(session.expiresAt) {
		return empty, errors.New("扫码会话已过期，请重新获取二维码")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	status, err := session.client.PollQRStatus(ctx, session.qrCode, session.pollBaseURL)
	if err != nil {
		return empty, err
	}
	switch status.Status {
	case "wait":
		return wechatResp.BindStatusResult{Status: "WAITING"}, nil
	case "scaned":
		return wechatResp.BindStatusResult{Status: "SCANNED"}, nil
	case "scaned_but_redirect":
		if status.RedirectHost != "" {
			session.pollBaseURL = "https://" + status.RedirectHost
		}
		return wechatResp.BindStatusResult{Status: "SCANNED"}, nil
	case "expired":
		qr, fetchErr := session.client.FetchQRCode(ctx)
		if fetchErr != nil {
			return empty, fmt.Errorf("刷新微信登录二维码: %w", fetchErr)
		}
		session.qrCode = qr.QRCode
		session.pollBaseURL = ilink.DefaultBaseURL
		return wechatResp.BindStatusResult{Status: "EXPIRED", QRCode: qr.QRCodeImgContent}, nil
	case "confirmed":
		bot, saveErr := s.saveConfirmedBot(session, status)
		if saveErr != nil {
			return empty, saveErr
		}
		result := wechatResp.BindStatusResult{Status: "CONFIRMED", Bot: &bot}
		session.result = &result
		time.AfterFunc(time.Minute, func() { s.removeSession(params.SessionID, session) })
		return result, nil
	default:
		return wechatResp.BindStatusResult{Status: strings.ToUpper(status.Status)}, nil
	}
}

func (s *BotService) Rename(params *wechatReq.RenameRequest) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("Bot ID 不能为空")
	}
	name := strings.TrimSpace(params.Name)
	if name == "" || len([]rune(name)) > 100 {
		return false, errors.New("Bot 名称不能为空且不能超过 100 个字符")
	}
	err := global.DB.Model(&wechatModel.Bot{}).Where("id = ?", params.ID).Update("name", name).Error
	return err == nil, err
}

func (s *BotService) Reconnect(params *wechatReq.BotIDRequest) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("Bot ID 不能为空")
	}
	if err := global.DB.Model(&wechatModel.Bot{}).Where("id = ?", params.ID).Updates(map[string]any{
		"enabled": true, "status": wechatModel.BotStatusConnecting, "last_error": "",
	}).Error; err != nil {
		return false, err
	}
	if err := s.runtime.StartBot(params.ID); err != nil {
		_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", params.ID).Updates(map[string]any{
			"status": wechatModel.BotStatusError, "last_error": truncateError(err),
		}).Error
		return false, err
	}
	return true, nil
}

func (s *BotService) SetEnabled(params *wechatReq.SetEnabledRequest) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("Bot ID 不能为空")
	}
	if !params.Enabled {
		s.runtime.StopBot(params.ID)
		err := global.DB.Model(&wechatModel.Bot{}).Where("id = ?", params.ID).Updates(map[string]any{
			"enabled": false, "status": wechatModel.BotStatusDisabled, "last_error": "",
		}).Error
		return err == nil, err
	}
	return s.Reconnect(&wechatReq.BotIDRequest{ID: params.ID})
}

func (s *BotService) Delete(params *wechatReq.BotIDRequest) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("Bot ID 不能为空")
	}
	s.runtime.StopBot(params.ID)
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("bot_record_id = ?", params.ID).Delete(&wechatModel.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("bot_record_id = ?", params.ID).Delete(&wechatModel.Contact{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&wechatModel.Bot{}, params.ID).Error
	})
	return err == nil, err
}

func (s *BotService) Contacts(params *wechatReq.ContactListParams) ([]wechatResp.ContactResult, error) {
	if params == nil || params.BotID == 0 {
		return nil, errors.New("Bot ID 不能为空")
	}
	var contacts []wechatModel.Contact
	if err := global.DB.Where("bot_record_id = ?", params.BotID).Order("last_active_at DESC").Limit(200).Find(&contacts).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	result := make([]wechatResp.ContactResult, 0, len(contacts))
	for _, contact := range contacts {
		result = append(result, wechatResp.ContactResult{
			ID: contact.ID, UserID: contact.UserID, LastMessage: contact.LastMessage,
			MessageCount: contact.MessageCount, LastActiveAt: contact.LastActiveAt,
			CanSend: now.Sub(time.UnixMilli(contact.LastActiveAt)) < sendWindow,
		})
	}
	return result, nil
}

func (s *BotService) Messages(params *wechatReq.MessagePageParams) (coreResponse.PageResult, error) {
	if params == nil || params.BotID == 0 || strings.TrimSpace(params.UserID) == "" {
		return coreResponse.PageResult{}, errors.New("Bot 和联系人不能为空")
	}
	var messages []wechatModel.Message
	query := global.DB.Model(&wechatModel.Message{}).
		Where("bot_record_id = ? AND user_id = ?", params.BotID, strings.TrimSpace(params.UserID)).
		Order("occurred_at DESC")
	return db.Page(query, params.GetPageInfo(), messages)
}

func (s *BotService) Send(params *wechatReq.SendMessageRequest) (wechatModel.Message, error) {
	var empty wechatModel.Message
	if params == nil || params.BotID == 0 || strings.TrimSpace(params.UserID) == "" {
		return empty, errors.New("Bot 和联系人不能为空")
	}
	content := strings.TrimSpace(params.Content)
	if content == "" || len([]byte(content)) > 4000 {
		return empty, errors.New("消息不能为空且不能超过 4000 字节")
	}
	userID := strings.TrimSpace(params.UserID)
	return s.send(params.BotID, userID, content)
}

func (s *BotService) WebhookCredentials(params *wechatReq.WebhookCredentialsParams) (wechatResp.WebhookCredentialsResult, error) {
	var result wechatResp.WebhookCredentialsResult
	if params == nil || params.Ctx == nil || params.Request.ID == 0 {
		return result, errors.New("Bot ID 不能为空")
	}
	s.credentialMu.Lock()
	defer s.credentialMu.Unlock()
	var bot wechatModel.Bot
	if err := global.DB.First(&bot, params.Request.ID).Error; err != nil {
		return result, err
	}
	var plaintextSecret string
	if strings.TrimSpace(bot.WebhookKey) == "" {
		key, err := s.uniqueWebhookKey()
		if err != nil {
			return result, err
		}
		bot.WebhookKey = key
	}
	if strings.TrimSpace(bot.WebhookSecret) == "" || params.Request.RotateSecret {
		secret, err := randomSessionID()
		if err != nil {
			return result, err
		}
		encrypted, err := s.vault.Encrypt(secret)
		if err != nil {
			return result, err
		}
		bot.WebhookSecret = encrypted
		plaintextSecret = secret
	}
	if err := global.DB.Model(&bot).Updates(map[string]any{
		"webhook_key": bot.WebhookKey, "webhook_secret": bot.WebhookSecret,
	}).Error; err != nil {
		return result, err
	}
	result.WebhookURL = fmt.Sprintf("%s/v1/apps/wechat-bot/webhook/%s", requestContextBaseURL(params.Ctx), bot.WebhookKey)
	result.WebhookSecret = plaintextSecret
	return result, nil
}

func (s *BotService) WebhookPush(params *wechatReq.WebhookPushParams) (wechatResp.WebhookPushResult, error) {
	var result wechatResp.WebhookPushResult
	if params == nil || strings.TrimSpace(params.WebhookKey) == "" {
		return result, ErrWebhookUnauthorized
	}
	var bot wechatModel.Bot
	if err := global.DB.Where("webhook_key = ?", strings.TrimSpace(params.WebhookKey)).First(&bot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, ErrWebhookUnauthorized
		}
		return result, err
	}
	secret, err := s.vault.Decrypt(bot.WebhookSecret)
	if err != nil {
		return result, err
	}
	if !verifyWebhookSignature(secret, params.Timestamp, params.Signature, time.Now()) {
		return result, ErrWebhookUnauthorized
	}
	if !bot.Enabled || bot.Status != wechatModel.BotStatusOnline {
		return result, errors.New("微信 Bot 当前不在线")
	}
	content := strings.TrimSpace(params.Request.Content)
	if content == "" || len([]byte(content)) > 4000 {
		return result, errors.New("消息不能为空且不能超过 4000 字节")
	}
	userID := strings.TrimSpace(params.Request.UserID)
	if userID == "" {
		var contact wechatModel.Contact
		cutoff := time.Now().Add(-sendWindow).UnixMilli()
		if err := global.DB.Where("bot_record_id = ? AND last_active_at > ?", bot.ID, cutoff).
			Order("last_active_at DESC, id DESC").First(&contact).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return result, errors.New("没有可发送的有效会话，请先让用户向 Bot 发送一条消息")
			}
			return result, err
		}
		userID = contact.UserID
	}
	message, err := s.send(bot.ID, userID, content)
	if err != nil {
		return result, err
	}
	result.Message = message
	return result, nil
}

func verifyWebhookSignature(secret, timestamp, provided string, now time.Time) bool {
	secret = strings.TrimSpace(secret)
	timestamp = strings.TrimSpace(timestamp)
	provided = strings.TrimSpace(provided)
	if secret == "" || timestamp == "" || provided == "" {
		return false
	}
	timestampMillis, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	nowMillis := now.UnixMilli()
	skewMillis := webhookTimestampSkew.Milliseconds()
	if timestampMillis < nowMillis-skewMillis || timestampMillis > nowMillis+skewMillis {
		return false
	}
	received, err := base64.StdEncoding.DecodeString(provided)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	return hmac.Equal(received, mac.Sum(nil))
}

func (s *BotService) send(botID snowflake.ID, userID, content string) (wechatModel.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	message, err := s.runtime.Send(ctx, botID, userID, content)
	if err != nil {
		return wechatModel.Message{}, err
	}
	if s.outboundHandler != nil {
		if mirrorErr := s.outboundHandler(botID, userID, content); mirrorErr != nil {
			// The WeChat message has already been accepted upstream. Do not return an
			// error that could make the operator retry and send a duplicate message.
			logWechatError("同步 Bot 管理端消息到 Agent 接管会话", mirrorErr)
		}
	}
	return message, nil
}

func (s *BotService) uniqueWebhookKey() (string, error) {
	for attempts := 0; attempts < 5; attempts++ {
		key, err := randomSessionID()
		if err != nil {
			return "", err
		}
		var count int64
		if err := global.DB.Model(&wechatModel.Bot{}).Where("webhook_key = ?", key).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return key, nil
		}
	}
	return "", errors.New("生成 Webhook 标识失败，请重试")
}

func (s *BotService) saveConfirmedBot(session *bindSession, status *ilink.QRStatusResponse) (wechatModel.Bot, error) {
	var bot wechatModel.Bot
	if status.ILinkBotID == "" || status.BotToken == "" {
		return bot, errors.New("微信扫码确认结果缺少 Bot 身份凭据")
	}
	encryptedToken, err := s.vault.Encrypt(status.BotToken)
	if err != nil {
		return bot, err
	}
	now := time.Now().UnixMilli()
	if session.targetBotID != 0 {
		if err = global.DB.First(&bot, session.targetBotID).Error; err != nil {
			return bot, err
		}
		bot.Name = session.name
		bot.BotID = status.ILinkBotID
		bot.BotToken = encryptedToken
		bot.BaseURL = status.BaseURL
		bot.ILinkUserID = status.ILinkUserID
		bot.Status = wechatModel.BotStatusConnecting
		bot.Enabled = true
		bot.SyncBuf = ""
		bot.LastError = ""
		bot.ConnectedAt = now
		err = global.DB.Model(&bot).Select(
			"name", "bot_id", "bot_token", "base_url", "i_link_user_id", "status", "enabled", "sync_buf", "last_error", "connected_at",
		).Updates(&bot).Error
	} else {
		if err = global.DB.Where("bot_id = ?", status.ILinkBotID).First(&bot).Error; err == nil {
			return wechatModel.Bot{}, errors.New("该微信 Bot 已存在，请在原账户上执行重新扫码")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return bot, err
		}
		name := session.name
		if name == "" {
			suffix := status.ILinkBotID
			if len(suffix) > 6 {
				suffix = suffix[len(suffix)-6:]
			}
			name = "微信 Bot " + suffix
		}
		bot = session.createdBy
		bot.Name = name
		bot.BotID = status.ILinkBotID
		bot.BotToken = encryptedToken
		bot.BaseURL = status.BaseURL
		bot.ILinkUserID = status.ILinkUserID
		bot.Status = wechatModel.BotStatusConnecting
		bot.Enabled = true
		bot.ConnectedAt = now
		err = global.DB.Create(&bot).Error
	}
	if err != nil {
		return bot, err
	}
	if err = s.runtime.StartBot(bot.ID); err != nil {
		_ = global.DB.Model(&bot).Updates(map[string]any{"status": wechatModel.BotStatusError, "last_error": truncateError(err)}).Error
		return bot, err
	}
	bot.Status = wechatModel.BotStatusConnecting
	bot.LastError = ""
	return bot, nil
}

func (s *BotService) removeSession(id string, expected *bindSession) {
	s.bindMu.Lock()
	if s.sessions[id] == expected {
		delete(s.sessions, id)
	}
	s.bindMu.Unlock()
}

func randomSessionID() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func requestContextBaseURL(c *fiber.Ctx) string {
	baseURL := strings.TrimRight(c.BaseURL(), "/")
	if index := strings.LastIndex(c.Path(), "/v1/"); index > 0 {
		baseURL += strings.TrimRight(c.Path()[:index], "/")
	}
	return baseURL
}
