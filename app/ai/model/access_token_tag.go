package model

import "apipig/core/api"

// AccessTokenTag 表示 API 密钥可使用的管理标签。
type AccessTokenTag struct {
	api.MODEL        // 通用主键、创建更新信息和软删除标记
	Name      string `gorm:"size:30;not null;index" json:"name"`                        // 标签名称
	Remark    string `gorm:"size:120" json:"remark"`                                    // 标签备注
	Sort      int    `gorm:"column:sort_order;type:int;not null;default:0" json:"sort"` // 排序值，越小越靠前
}

// TableName 返回 API 密钥标签表名。
func (AccessTokenTag) TableName() string { return "ap_ai_access_token_tag" }
