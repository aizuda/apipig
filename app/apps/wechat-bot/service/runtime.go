package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	aiService "apipig/app/ai/service"
	wechatModel "apipig/app/apps/wechat-bot/model"
	"apipig/core/api"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	ilink "github.com/openilink/openilink-sdk-go"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

const sendWindow = 24 * time.Hour

type botConnection struct {
	client *ilink.Client
	cancel context.CancelFunc
	sendMu sync.Mutex
}

// Runtime owns the in-process long-poll connection for every enabled Bot.
type Runtime struct {
	vault  aiService.CredentialVault
	mu     sync.RWMutex
	root   context.Context
	cancel context.CancelFunc
	conns  map[snowflake.ID]*botConnection
	wg     sync.WaitGroup
}

func newRuntime(vault aiService.CredentialVault) *Runtime {
	return &Runtime{vault: vault, conns: make(map[snowflake.ID]*botConnection)}
}

func (r *Runtime) StartAll() {
	r.mu.Lock()
	if r.root == nil {
		r.root, r.cancel = context.WithCancel(context.Background())
	}
	r.mu.Unlock()

	var bots []wechatModel.Bot
	if err := global.DB.Where("enabled = ?", true).Find(&bots).Error; err != nil {
		logWechatError("load WeChat Bots", err)
		return
	}
	for i := range bots {
		if err := r.StartBot(bots[i].ID); err != nil {
			logWechatError("start WeChat Bot", err)
			_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", bots[i].ID).Updates(map[string]any{
				"status": wechatModel.BotStatusError, "last_error": truncateError(err),
			}).Error
		}
	}
}

func (r *Runtime) StartBot(id snowflake.ID) error {
	var bot wechatModel.Bot
	if err := global.DB.First(&bot, id).Error; err != nil {
		return err
	}
	if !bot.Enabled {
		return errors.New("微信 Bot 已停用")
	}
	token, err := r.vault.Decrypt(bot.BotToken)
	if err != nil {
		return fmt.Errorf("解密 Bot Token: %w", err)
	}
	options := make([]ilink.Option, 0, 1)
	if strings.TrimSpace(bot.BaseURL) != "" {
		options = append(options, ilink.WithBaseURL(bot.BaseURL))
	}
	client := ilink.NewClient(token, options...)

	r.mu.Lock()
	if r.root == nil {
		r.root, r.cancel = context.WithCancel(context.Background())
	}
	if old := r.conns[id]; old != nil {
		old.cancel()
	}
	ctx, cancel := context.WithCancel(r.root)
	conn := &botConnection{client: client, cancel: cancel}
	r.conns[id] = conn
	r.mu.Unlock()

	now := time.Now().UnixMilli()
	_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", id).Updates(map[string]any{
		"status": wechatModel.BotStatusConnecting, "last_error": "", "connected_at": now,
	}).Error

	r.wg.Add(1)
	go r.monitor(ctx, &bot, conn)
	return nil
}

func (r *Runtime) monitor(ctx context.Context, bot *wechatModel.Bot, conn *botConnection) {
	defer r.wg.Done()
	_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", bot.ID).Updates(map[string]any{
		"status": wechatModel.BotStatusOnline, "last_error": "",
	}).Error

	err := conn.client.Monitor(ctx, func(message ilink.WeixinMessage) {
		if saveErr := r.storeInbound(bot.ID, message); saveErr != nil {
			logWechatError("store inbound WeChat message", saveErr)
		}
	}, &ilink.MonitorOptions{
		InitialBuf: bot.SyncBuf,
		OnBufUpdate: func(buf string) {
			if err := global.DB.Model(&wechatModel.Bot{}).Where("id = ?", bot.ID).Update("sync_buf", buf).Error; err != nil {
				logWechatError("update WeChat Bot cursor", err)
			}
		},
		OnError: func(pollErr error) {
			_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", bot.ID).Update("last_error", truncateError(pollErr)).Error
		},
		OnSessionExpired: func() {
			_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", bot.ID).Updates(map[string]any{
				"status": wechatModel.BotStatusSessionExpired, "last_error": "iLink 会话已过期，请重新扫码绑定",
			}).Error
		},
	})

	r.mu.Lock()
	current := r.conns[bot.ID]
	if current == conn {
		delete(r.conns, bot.ID)
	}
	r.mu.Unlock()
	if current != conn {
		return
	}
	var latest wechatModel.Bot
	if global.DB.Select("status", "enabled").First(&latest, bot.ID).Error != nil || !latest.Enabled || latest.Status == wechatModel.BotStatusSessionExpired {
		return
	}
	status := wechatModel.BotStatusOffline
	lastError := ""
	if err != nil && !errors.Is(err, context.Canceled) {
		status = wechatModel.BotStatusError
		lastError = truncateError(err)
	}
	_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", bot.ID).Updates(map[string]any{
		"status": status, "last_error": lastError,
	}).Error
}

func (r *Runtime) StopBot(id snowflake.ID) {
	r.mu.Lock()
	if conn := r.conns[id]; conn != nil {
		conn.cancel()
		delete(r.conns, id)
	}
	r.mu.Unlock()
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
	}
	r.conns = make(map[snowflake.ID]*botConnection)
	r.root = nil
	r.cancel = nil
	r.mu.Unlock()
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runtime) Send(ctx context.Context, botID snowflake.ID, userID, content string) (wechatModel.Message, error) {
	var result wechatModel.Message
	r.mu.RLock()
	conn := r.conns[botID]
	r.mu.RUnlock()
	if conn == nil {
		return result, errors.New("微信 Bot 当前未连接")
	}
	var contact wechatModel.Contact
	if err := global.DB.Where("bot_record_id = ? AND user_id = ?", botID, userID).First(&contact).Error; err != nil {
		return result, errors.New("未找到联系人上下文，请先让该用户向 Bot 发送一条消息")
	}
	if time.Since(time.UnixMilli(contact.LastActiveAt)) >= sendWindow {
		return result, errors.New("联系人会话已超过 24 小时，请先让该用户向 Bot 发送一条消息")
	}
	contextToken, err := r.vault.Decrypt(contact.ContextToken)
	if err != nil {
		return result, fmt.Errorf("解密联系人上下文: %w", err)
	}
	conn.sendMu.Lock()
	clientID, err := conn.client.SendText(ctx, userID, content, contextToken)
	conn.sendMu.Unlock()
	if err != nil {
		return result, err
	}
	now := time.Now().UnixMilli()
	result = wechatModel.Message{
		MODEL: api.MODEL{ID: db.GetId(), CreatedAt: now}, BotRecordID: botID,
		ExternalMessageID: clientID, Direction: "outbound", UserID: userID,
		Content: content, ContentType: "text", OccurredAt: now,
	}
	if err = global.DB.Create(&result).Error; err != nil {
		return result, err
	}
	_ = global.DB.Model(&wechatModel.Bot{}).Where("id = ?", botID).Updates(map[string]any{
		"message_count": clause.Expr{SQL: "message_count + 1"}, "last_message_at": now,
	}).Error
	return result, nil
}

func (r *Runtime) storeInbound(botID snowflake.ID, message ilink.WeixinMessage) error {
	now := message.CreateTimeMs
	if now <= 0 {
		now = time.Now().UnixMilli()
	}
	content := strings.TrimSpace(ilink.ExtractText(&message))
	contentType := messageContentType(message)
	if content == "" {
		content = "[" + contentType + "]"
	}
	externalID := strconv.FormatInt(message.MessageID, 10)
	if message.MessageID == 0 {
		externalID = fmt.Sprintf("seq:%d:%d", message.Seq, now)
	}
	record := wechatModel.Message{
		MODEL: api.MODEL{ID: db.GetId(), CreatedAt: time.Now().UnixMilli()}, BotRecordID: botID,
		ExternalMessageID: externalID, Direction: "inbound", UserID: message.FromUserID,
		Content: content, ContentType: contentType, SessionID: message.SessionID,
		GroupID: message.GroupID, OccurredAt: now,
	}
	created := global.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
	if created.Error != nil {
		return created.Error
	}
	if message.ContextToken != "" && message.FromUserID != "" {
		encryptedToken, err := r.vault.Encrypt(message.ContextToken)
		if err != nil {
			return err
		}
		contact := wechatModel.Contact{
			MODEL: api.MODEL{ID: db.GetId(), CreatedAt: time.Now().UnixMilli()}, BotRecordID: botID,
			UserID: message.FromUserID, ContextToken: encryptedToken, LastMessage: truncateText(content, 500),
			MessageCount: 1, LastActiveAt: now,
		}
		contactUpdates := map[string]any{
			"context_token": encryptedToken, "last_message": contact.LastMessage,
			"last_active_at": now,
		}
		if created.RowsAffected > 0 {
			contactUpdates["message_count"] = clause.Expr{SQL: "message_count + 1"}
		}
		if err = global.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "bot_record_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(contactUpdates),
		}).Create(&contact).Error; err != nil {
			return err
		}
	}
	updates := map[string]any{"status": wechatModel.BotStatusOnline, "last_error": ""}
	if created.RowsAffected > 0 {
		updates["message_count"] = clause.Expr{SQL: "message_count + 1"}
		updates["last_message_at"] = now
	}
	return global.DB.Model(&wechatModel.Bot{}).Where("id = ?", botID).Updates(updates).Error
}

func messageContentType(message ilink.WeixinMessage) string {
	for _, item := range message.ItemList {
		switch item.Type {
		case ilink.ItemText:
			return "text"
		case ilink.ItemImage:
			return "image"
		case ilink.ItemVoice:
			return "voice"
		case ilink.ItemFile:
			return "file"
		case ilink.ItemVideo:
			return "video"
		}
	}
	return "unknown"
}

func truncateText(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	return truncateText(err.Error(), 500)
}

func logWechatError(message string, err error) {
	if global.LOG != nil {
		global.LOG.Error(message, zap.Error(err))
	}
}
