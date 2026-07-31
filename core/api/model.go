package api

import (
	"apipig/toolkit/snowflake"
)

type MODEL struct {
	ID        snowflake.ID `gorm:"type:bigint;primaryKey" json:"id,omitempty" swaggertype:"string"`      // 主键ID
	CreatedId snowflake.ID `gorm:"type:bigint;not null" json:"createdId,omitempty" swaggertype:"string"` // 创建人ID
	CreatedBy string       `gorm:"not null;size:50" json:"createdBy,omitempty"`                          // 创建人
	CreatedAt int64        `gorm:"type:bigint;not null;autoCreateTime:milli" json:"createdAt,omitempty"` // 创建时间
	UpdatedBy string       `gorm:"size:50" json:"updatedBy,omitempty"`                                   // 更新人
	UpdatedAt int64        `gorm:"type:bigint;autoUpdateTime:milli" json:"updatedAt,omitempty"`          // 更新时间
	DeletedAt DeletedAt    `gorm:"type:bigint;softDelete:milli" json:"-"`                                // 删除时间
}

func NormalDisable(status uint) uint {
	if status == 2 {
		// 2、禁用
		return 2
	}
	// 1、正常
	return 1
}
