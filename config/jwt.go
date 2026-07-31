package config

import (
	"apipig/toolkit"
)

type JWT struct {
	SigningKey  string `mapstructure:"signing-key" json:"signingKey" yaml:"signing-key"`    // jwt签名
	ExpiresTime int64  `mapstructure:"expires-time" json:"expiresTime" yaml:"expires-time"` // 过期时间（分钟）
	IgnoreIp    bool   `mapstructure:"check-ip" json:"checkIp" yaml:"check-ip"`             // 忽略IP校验，默认验证
	PrivateKey  string `mapstructure:"private-key" json:"privateKey" yaml:"private-key"`    // RSA私钥
	PublicKey   string `mapstructure:"public-key" json:"publicKey" yaml:"public-key"`       // RSA公钥
}

var expiresTime int64

func (d *JWT) GetExpiresTime() int64 {
	if expiresTime == 0 {
		expiresTime = d.ExpiresTime
	}
	return toolkit.GetNowLocal().UnixMilli() - expiresTime
}
