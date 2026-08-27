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
	"apipig/global"
	"apipig/toolkit/snowflake"
)

// ProxyService 负责代理节点表的数据校验、持久化和查询。
type ProxyService struct {
	gateway *GatewayService
	vault   CredentialVault
	store   AIStore
}

func (s *ProxyService) dependencies() (*GatewayService, CredentialVault) {
	vault := s.vault
	if vault == nil {
		vault = newAESCredentialVault(func() string { return global.CONFIG.AI.EncryptionKey })
	}
	return s.gateway, vault
}

func (s *ProxyService) persistence() AIStore {
	return resolveAIStore(s.store)
}

// Save 创建或更新代理节点。
func (s *ProxyService) Save(params *aiReq.ProxySaveParams) (bool, error) {
	if params == nil || params.Proxy == nil {
		return false, errors.New("代理参数不能为空")
	}
	m := params.Proxy
	normalizeProxy(m)
	if err := validateProxy(m); err != nil {
		return false, err
	}
	if m.ID == 0 {
		_, vault := s.dependencies()
		storedPassword, err := vault.Encrypt(m.Password)
		if err != nil {
			return false, err
		}
		m.Password = storedPassword
		m.MODEL = db.NewModel(params.Ctx)
		success, err := s.persistence().Create(m)
		if success {
			if gateway, _ := s.dependencies(); gateway != nil {
				gateway.invalidateRouteTargets()
			}
		}
		return success, err
	}
	preserveCredential := isMaskedCredential(m.Password)
	if preserveCredential {
		var existing model.Proxy
		if err := s.persistence().GetByID(&existing, m.ID); err != nil {
			return false, err
		}
		m.Password = existing.Password
	}
	_, vault := s.dependencies()
	if !preserveCredential || vault.Enabled() {
		storedPassword, err := vault.Encrypt(m.Password)
		if err != nil {
			return false, err
		}
		m.Password = storedPassword
	}
	success, err := s.persistence().Update(m)
	if success {
		if gateway, _ := s.dependencies(); gateway != nil {
			gateway.invalidateRouteTargets()
		}
	}
	return success, err
}

// ChangeStatus 切换代理节点启用、禁用状态。
func (s *ProxyService) ChangeStatus(params *aiReq.StatusChangeParams) (bool, error) {
	success, err := changeResourceStatus(s.persistence(), model.Proxy{}, "IP 代理", params)
	if success && s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
	return success, err
}

// Delete 根据 ID 集合批量删除代理节点。
func (s *ProxyService) Delete(idsReq *request.IdsReq) (bool, error) {
	if err := validateAIBulkIDs(idsReq, "请选择要删除的代理"); err != nil {
		return false, err
	}
	var channelCount int64
	if err := s.persistence().Query(model.Channel{}).Where("proxy_id IN ?", idsReq.Ids).Count(&channelCount).Error; err != nil {
		return false, err
	}
	if channelCount > 0 {
		return false, errors.New("代理仍有关联渠道，不能删除")
	}
	success, err := s.persistence().DeleteByIDs(model.Proxy{}, idsReq.Ids)
	if success {
		if gateway, _ := s.dependencies(); gateway != nil {
			gateway.invalidateRouteTargets()
		}
	}
	return success, err
}

// Get 根据 ID 查询代理节点。
func (s *ProxyService) Get(id snowflake.ID) (m model.Proxy, err error) {
	err = s.persistence().GetByID(&m, id)
	if err == nil && m.Password != "" {
		m.Password = maskedCredential
	}
	return
}

// Page 按名称、区域和状态分页查询代理节点。
func (s *ProxyService) Page(params *aiReq.ProxyPageParams) (response.PageResult, error) {
	query := s.persistence().Query(model.Proxy{})
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where(
				"(name LIKE ? OR host LIKE ? OR region LIKE ? OR scheme LIKE ?)",
				likeKeyword, likeKeyword, likeKeyword, likeKeyword,
			)
		}
		if params.Name != "" {
			query = query.Where("name LIKE ?", "%"+params.Name+"%")
		}
		if params.Region != "" {
			query = query.Where("region LIKE ?", "%"+params.Region+"%")
		}
		if params.Status > 0 {
			query = query.Where("status = ?", params.Status)
		}
	}
	var arr []model.Proxy
	result, err := s.persistence().Page(query.Order("created_at DESC"), pageInfo(params), arr)
	if err != nil {
		return result, err
	}
	return maskPageRecords(result, func(item *model.Proxy) {
		if item.Password != "" {
			item.Password = maskedCredential
		}
	}), nil
}

// normalizeProxy 统一清理代理节点字段并补齐默认值。
func normalizeProxy(m *model.Proxy) {
	m.Name = strings.TrimSpace(m.Name)
	m.Scheme = strings.ToLower(defaultString(strings.TrimSpace(m.Scheme), "http"))
	m.Host = strings.Trim(strings.TrimSpace(m.Host), "[]")
	m.Username = strings.TrimSpace(m.Username)
	m.Region = strings.TrimSpace(m.Region)
	m.Remark = strings.TrimSpace(m.Remark)
	m.Status = api.NormalDisable(m.Status)
}
