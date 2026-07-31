package service

import (
	"apipig/app/sys/model"
	sysReq "apipig/app/sys/model/request"
	"apipig/core/api/request"
	"apipig/core/cache"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"
	"gorm.io/gorm"
)

type ResourceApiService struct {
}

func (s *ResourceApiService) getDB() *gorm.DB {
	return db.GetDB(model.ResourceApi{})
}

func (s *ResourceApiService) Create(params *sysReq.ResourceApiCreateParams) (*snowflake.ID, error) {
	ra := &model.ResourceApi{
		MODEL:      db.NewModel(params.Ctx),
		ResourceId: params.ResourceId,
		Method:     params.Method,
		Url:        params.Url,
	}
	err := global.DB.Create(ra).Error
	if err == nil {
		s.resetRbacCache()
		return &ra.ID, err
	}
	return nil, err
}

func (s *ResourceApiService) Delete(idsReq *request.IdsReq) (bool, error) {
	s.resetRbacCache()
	return db.DeleteByIds(model.ResourceApi{}, idsReq.Ids)
}
func (s *ResourceApiService) Update(m *model.ResourceApi) (bool, error) {
	s.resetRbacCache()
	return db.Update(m)
}

// 重置 Rbac 缓存
func (s *ResourceApiService) resetRbacCache() {
	_ = cache.RbacCache.Reset()
}

func (s *ResourceApiService) ListByResourceId(params *sysReq.ResourceApiParams) (arr []model.ResourceApi, err error) {
	t := s.getDB()
	if len(params.Url) > 0 {
		t.Where("url like ?", "%"+params.Url+"%")
	}
	err = t.Where("resource_id = ?", params.ResourceId).Find(&arr).Error
	return
}
