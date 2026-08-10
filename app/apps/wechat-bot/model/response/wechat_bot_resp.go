package response

import (
	wechatModel "apipig/app/apps/wechat-bot/model"
	"apipig/toolkit/snowflake"
)

type BindStartResult struct {
	SessionID string `json:"sessionId"`
	QRCode    string `json:"qrCode"`
}

type BindStatusResult struct {
	Status string           `json:"status"`
	QRCode string           `json:"qrCode,omitempty"`
	Bot    *wechatModel.Bot `json:"bot,omitempty"`
}

type ContactResult struct {
	ID           snowflake.ID `json:"id" swaggertype:"string"`
	UserID       string       `json:"userId"`
	LastMessage  string       `json:"lastMessage"`
	MessageCount int64        `json:"messageCount"`
	LastActiveAt int64        `json:"lastActiveAt"`
	CanSend      bool         `json:"canSend"`
}

type WebhookCredentialsResult struct {
	WebhookURL    string `json:"webhookUrl"`
	WebhookSecret string `json:"webhookSecret,omitempty"`
}

type WebhookPushResult struct {
	Message wechatModel.Message `json:"message"`
}
