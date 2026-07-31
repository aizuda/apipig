package service

import (
	"strings"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/core/api"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/toolkit"
)

// CallLogService 负责调用日志表的写入和分页查询。
type CallLogService struct{ store AIStore }

func (s *CallLogService) persistence() AIStore {
	return resolveAIStore(s.store)
}

// Create 写入一条网关调用审计日志。
func (s *CallLogService) Create(logRecord model.CallLog) error {
	logRecord.MODEL = api.MODEL{
		ID:        db.GetId(),
		CreatedAt: toolkit.GetNowUnixMilli(),
		CreatedBy: "ai-gateway",
	}
	_, err := s.persistence().Create(&logRecord)
	return err
}

// Page 按模型、关联对象、调用结果和时间范围分页查询调用日志。
func (s *CallLogService) Page(params *aiReq.CallLogPageParams) (response.PageResult, error) {
	query := s.persistence().Query(model.CallLog{})
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where(
				"(request_id LIKE ? OR model LIKE ? OR path LIKE ? OR client_ip LIKE ? OR error_message LIKE ?)",
				likeKeyword, likeKeyword, likeKeyword, likeKeyword, likeKeyword,
			)
		}
		if params.Model != "" {
			query = query.Where("model LIKE ?", "%"+params.Model+"%")
		}
		if params.ProviderID > 0 {
			query = query.Where("provider_id = ?", params.ProviderID)
		}
		if params.ChannelID > 0 {
			query = query.Where("channel_id = ?", params.ChannelID)
		}
		if params.TokenID > 0 {
			query = query.Where("access_token_id = ?", params.TokenID)
		}
		if params.Success > 0 {
			query = query.Where("success = ?", params.Success)
		}
		if params.StartAt > 0 {
			query = query.Where("created_at >= ?", params.StartAt)
		}
		if params.EndAt > 0 {
			query = query.Where("created_at <= ?", params.EndAt)
		}
	}
	var arr []model.CallLog
	return s.persistence().Page(query.Order("created_at DESC"), pageInfo(params), arr)
}
