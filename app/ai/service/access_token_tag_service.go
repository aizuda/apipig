package service

import (
	"errors"
	"strings"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/core/api/request"
	"apipig/core/db"
)

// AccessTokenTagService 负责 API 密钥标签的数据校验、持久化和排序。
type AccessTokenTagService struct{ store AIStore }

func (s *AccessTokenTagService) persistence() AIStore {
	return resolveAIStore(s.store)
}

// Save 创建或更新 API 密钥标签。
func (s *AccessTokenTagService) Save(params *aiReq.AccessTokenTagSaveParams) (bool, error) {
	if params == nil || params.Tag == nil {
		return false, errors.New("API 密钥标签参数不能为空")
	}
	tag := params.Tag
	tag.Name = strings.TrimSpace(tag.Name)
	tag.Remark = strings.TrimSpace(tag.Remark)
	if tag.Name == "" {
		return false, errors.New("标签名称不能为空")
	}
	if len([]rune(tag.Name)) > 30 {
		return false, errors.New("标签名称不能超过 30 个字符")
	}
	if len([]rune(tag.Remark)) > 120 {
		return false, errors.New("标签备注不能超过 120 个字符")
	}

	duplicateQuery := s.persistence().Query(model.AccessTokenTag{}).Where("LOWER(name) = ?", strings.ToLower(tag.Name))
	if tag.ID != 0 {
		duplicateQuery = duplicateQuery.Where("id <> ?", tag.ID)
	}
	var duplicateCount int64
	if err := duplicateQuery.Count(&duplicateCount).Error; err != nil {
		return false, err
	}
	if duplicateCount > 0 {
		return false, errors.New("标签名称已存在")
	}

	if tag.ID == 0 {
		if tag.Sort <= 0 {
			var maxSort int
			if err := s.persistence().Query(model.AccessTokenTag{}).Select("COALESCE(MAX(sort_order), 0)").Scan(&maxSort).Error; err != nil {
				return false, err
			}
			tag.Sort = maxSort + 1
		}
		tag.MODEL = db.NewModel(params.Ctx)
		return s.persistence().Create(tag)
	}

	var existing model.AccessTokenTag
	if err := s.persistence().GetByID(&existing, tag.ID); err != nil {
		return false, err
	}
	if tag.Sort <= 0 {
		tag.Sort = existing.Sort
	}
	tag.MODEL = existing.MODEL
	return s.persistence().Update(tag)
}

// Delete 批量删除 API 密钥标签。
func (s *AccessTokenTagService) Delete(idsReq *request.IdsReq) (bool, error) {
	if idsReq == nil || len(idsReq.Ids) == 0 {
		return false, errors.New("请选择要删除的标签")
	}
	var success bool
	err := s.persistence().Transaction(func(store AIStore) error {
		if err := store.Query(model.AccessTokenTagRelation{}).
			Where("tag_id IN ?", idsReq.Ids).
			Delete(&model.AccessTokenTagRelation{}).Error; err != nil {
			return err
		}
		result := store.Query(model.AccessTokenTag{}).
			Unscoped().
			Where("id IN ?", idsReq.Ids).
			Delete(&model.AccessTokenTag{})
		success = result.Error == nil
		return result.Error
	})
	return success && err == nil, err
}

// List 按自定义顺序查询全部 API 密钥标签。
func (s *AccessTokenTagService) List(_ *request.Empty) (tags []model.AccessTokenTag, err error) {
	err = s.persistence().Query(model.AccessTokenTag{}).Order("sort_order ASC, created_at ASC").Find(&tags).Error
	return
}

// Sort 保存 API 密钥标签的完整排序结果。
func (s *AccessTokenTagService) Sort(params *aiReq.AccessTokenTagSortParams) (bool, error) {
	if params == nil || len(params.IDs) == 0 {
		return false, errors.New("标签排序不能为空")
	}
	seen := make(map[string]struct{}, len(params.IDs))
	for _, id := range params.IDs {
		key := id.String()
		if _, exists := seen[key]; exists {
			return false, errors.New("标签排序包含重复项")
		}
		seen[key] = struct{}{}
	}

	var count int64
	if err := s.persistence().Query(model.AccessTokenTag{}).Where("id IN ?", params.IDs).Count(&count).Error; err != nil {
		return false, err
	}
	if count != int64(len(params.IDs)) {
		return false, errors.New("标签排序数据已变化，请刷新后重试")
	}
	var total int64
	if err := s.persistence().Query(model.AccessTokenTag{}).Count(&total).Error; err != nil {
		return false, err
	}
	if total != int64(len(params.IDs)) {
		return false, errors.New("请提交完整的标签排序")
	}

	err := s.persistence().Transaction(func(store AIStore) error {
		for index, id := range params.IDs {
			if err := store.Query(model.AccessTokenTag{}).
				Where("id = ?", id).
				Update("sort_order", index+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err == nil, err
}
