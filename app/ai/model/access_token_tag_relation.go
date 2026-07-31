package model

import (
	"apipig/toolkit/snowflake"
)

// AccessTokenTagRelation 表示 API 密钥与标签之间的多对多关联。
type AccessTokenTagRelation struct {
	AccessTokenID snowflake.ID `gorm:"type:bigint;primaryKey;index" json:"accessTokenId" swaggertype:"string"` // API 密钥 ID
	TagID         snowflake.ID `gorm:"type:bigint;primaryKey;index" json:"tagId" swaggertype:"string"`         // 标签 ID
	CreatedAt     int64        `gorm:"type:bigint;not null;autoCreateTime:milli" json:"createdAt,omitempty"`   // 关联创建时间，毫秒时间戳
}

// TableName 返回 API 密钥标签关联表名。
func (AccessTokenTagRelation) TableName() string { return "ap_ai_access_token_tag_relation" }
