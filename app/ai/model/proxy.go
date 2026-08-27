package model

import "apipig/core/api"

// Proxy 表示渠道转发请求时可选用的代理节点。
//
// 代理节点用于固定出口 IP、跨地域访问或隔离不同渠道账号的网络出口。
type Proxy struct {
	api.MODEL        // 通用主键、创建更新信息和软删除标记
	Name      string `gorm:"size:80;not null;index" json:"name"`             // 代理名称
	Scheme    string `gorm:"size:20;not null;default:http" json:"scheme"`    // 代理协议：http、https、socks5
	Host      string `gorm:"size:120;not null" json:"host"`                  // 代理主机
	Port      int    `gorm:"type:int;not null" json:"port"`                  // 代理端口
	Username  string `gorm:"size:120" json:"username"`                       // 认证用户名
	Password  string `gorm:"size:2048" json:"password"`                      // 认证密码，持久化时加密保存
	Region    string `gorm:"size:80" json:"region"`                          // 代理所在区域
	Status    uint   `gorm:"type:smallint;not null;default:1" json:"status"` // 状态：1 启用，2 禁用
	Remark    string `gorm:"size:255" json:"remark"`                         // 备注
}

// TableName 返回代理节点表名。
func (Proxy) TableName() string { return "ap_ai_proxy" }
