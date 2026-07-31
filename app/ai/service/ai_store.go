package service

import (
	"apipig/core/api/request"
	"apipig/core/api/response"
	coredb "apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

// AIStore 定义 AI 管理域所需的最小持久化能力。
type AIStore interface {
	Create(value any) (bool, error)
	Update(value any) (bool, error)
	DeleteByIDs(value any, ids any) (bool, error)
	GetByID(dest any, id snowflake.ID) error
	Query(value any) *gorm.DB
	Page(query *gorm.DB, pageInfo request.PageInfo, records any) (response.PageResult, error)
	Transaction(fn func(AIStore) error) error
}

type gormAIStore struct {
	dbProvider func() *gorm.DB
}

func newGormAIStore(dbProvider func() *gorm.DB) AIStore {
	return &gormAIStore{dbProvider: dbProvider}
}

func (s *gormAIStore) database() *gorm.DB {
	if s == nil || s.dbProvider == nil {
		return nil
	}
	return s.dbProvider()
}

func (s *gormAIStore) Create(value any) (bool, error) {
	err := s.database().Create(value).Error
	return err == nil, err
}

func (s *gormAIStore) Update(value any) (bool, error) {
	err := s.database().Updates(value).Error
	return err == nil, err
}

func (s *gormAIStore) DeleteByIDs(value any, ids any) (bool, error) {
	err := s.database().Where("id IN ?", ids).Delete(&value).Error
	return err == nil, err
}

func (s *gormAIStore) GetByID(dest any, id snowflake.ID) error {
	return s.database().First(dest, id).Error
}

func (s *gormAIStore) Query(value any) *gorm.DB {
	return s.database().Model(value)
}

func (s *gormAIStore) Page(query *gorm.DB, pageInfo request.PageInfo, records any) (response.PageResult, error) {
	return coredb.Page(query, pageInfo, records)
}

func (s *gormAIStore) Transaction(fn func(AIStore) error) error {
	return s.database().Transaction(func(tx *gorm.DB) error {
		return fn(newGormAIStore(func() *gorm.DB { return tx }))
	})
}

func resolveAIStore(store AIStore) AIStore {
	if store != nil {
		return store
	}
	return newGormAIStore(func() *gorm.DB { return global.DB })
}
