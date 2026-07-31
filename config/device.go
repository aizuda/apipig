package config

import (
	"apipig/toolkit"
)

type Device struct {
	ProductNumPrefix string `mapstructure:"product-num-prefix" json:"productNumPrefix" yaml:"product-num-prefix"` // 产品编码前缀
	CheckOfflineCron string `mapstructure:"check-offline-cron" json:"checkOfflineCron" yaml:"check-offline-cron"` // 设备离线检查时间单位
	CheckOfflineTime int64  `mapstructure:"check-offline-time" json:"checkOfflineTime" yaml:"check-offline-time"` // 检查设备离线时长（分钟）
	DataRelayCron    string `mapstructure:"data-relay-cron" json:"dataRelayCron" yaml:"data-relay-cron"`          // 设备数据转发时间单位
	DataRelayCount   uint   `mapstructure:"data-relay-count" json:"dataRelayCount" yaml:"data-relay-count"`       // 设备数据转发失败最大执行次数
}

var checkOfflineTime int64

func (d *Device) GetCheckOfflineTime() int64 {
	if checkOfflineTime == 0 {
		checkOfflineTime = d.CheckOfflineTime
	}
	return toolkit.GetNowLocal().UnixMilli() - checkOfflineTime
}
