package service

import (
	"errors"
	"strings"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/toolkit/snowflake"
)

// ProviderService 负责供应商表的数据校验、持久化和查询。
type ProviderService struct {
	gateway *GatewayService
	store   AIStore
}

func (s *ProviderService) persistence() AIStore {
	return resolveAIStore(s.store)
}

func (s *ProviderService) invalidateRoutes() {
	if s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
}

// Save 创建或更新供应商配置。
func (s *ProviderService) Save(params *aiReq.ProviderSaveParams) (bool, error) {
	if params == nil || params.Provider == nil {
		return false, errors.New("供应商参数不能为空")
	}
	m := params.Provider
	normalizeProvider(m)
	if err := validateProvider(m); err != nil {
		return false, err
	}
	if m.ID == 0 {
		m.MODEL = db.NewModel(params.Ctx)
		success, err := s.persistence().Create(m)
		if success {
			s.invalidateRoutes()
		}
		return success, err
	}
	success, err := s.persistence().Update(m)
	if success {
		s.invalidateRoutes()
	}
	return success, err
}

// ChangeStatus 切换供应商启用、禁用状态。
func (s *ProviderService) ChangeStatus(params *aiReq.StatusChangeParams) (bool, error) {
	success, err := changeResourceStatus(s.persistence(), model.Provider{}, "供应商", params)
	if success && s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
	return success, err
}

// Delete 根据 ID 集合批量删除供应商配置。
func (s *ProviderService) Delete(idsReq *request.IdsReq) (bool, error) {
	if err := validateAIBulkIDs(idsReq, "请选择要删除的供应商"); err != nil {
		return false, err
	}
	var channelCount int64
	if err := s.persistence().Query(model.Channel{}).Where("provider_id IN ?", idsReq.Ids).Count(&channelCount).Error; err != nil {
		return false, err
	}
	if channelCount > 0 {
		return false, errors.New("供应商仍有关联渠道，不能删除")
	}
	success, err := s.persistence().DeleteByIDs(model.Provider{}, idsReq.Ids)
	if success {
		s.invalidateRoutes()
	}
	return success, err
}

// Get 根据 ID 查询供应商配置。
func (s *ProviderService) Get(id snowflake.ID) (m model.Provider, err error) {
	err = s.persistence().GetByID(&m, id)
	return
}

// List 查询所有启用状态的供应商配置。
func (s *ProviderService) List(_ *request.Empty) (arr []model.Provider, err error) {
	err = s.persistence().Query(model.Provider{}).Where("status = ?", gatewayStatusNormal).Order("created_at DESC").Find(&arr).Error
	return
}

// Page 按名称、编码、协议和状态分页查询供应商配置。
func (s *ProviderService) Page(params *aiReq.ProviderPageParams) (response.PageResult, error) {
	query := s.persistence().Query(model.Provider{})
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where(
				"(name LIKE ? OR code LIKE ? OR protocol LIKE ? OR models LIKE ?)",
				likeKeyword, likeKeyword, likeKeyword, likeKeyword,
			)
		}
		if params.Name != "" {
			query = query.Where("name LIKE ?", "%"+params.Name+"%")
		}
		if params.Code != "" {
			query = query.Where("code LIKE ?", "%"+params.Code+"%")
		}
		if params.Protocol != "" {
			query = query.Where("protocol = ?", params.Protocol)
		}
		if params.Status > 0 {
			query = query.Where("status = ?", params.Status)
		}
	}
	var arr []model.Provider
	return s.persistence().Page(query.Order("created_at DESC"), pageInfo(params), arr)
}

// normalizeProvider 统一清理供应商字段并补齐默认值。
func normalizeProvider(m *model.Provider) {
	m.Name = strings.TrimSpace(m.Name)
	m.Code = strings.TrimSpace(m.Code)
	m.Icon = strings.ToLower(strings.TrimSpace(m.Icon))
	m.Protocol = strings.ToLower(defaultString(strings.TrimSpace(m.Protocol), "openai"))
	m.BaseURL = strings.TrimRight(strings.TrimSpace(m.BaseURL), "/")
	m.Models = normalizeModels(m.Models)
	m.Remark = strings.TrimSpace(m.Remark)
	if m.TimeoutMs <= 0 {
		m.TimeoutMs = 60000
	}
	m.Status = api.NormalDisable(m.Status)
}
