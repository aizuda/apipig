package config

type System struct {
	Port          int    `mapstructure:"port" json:"port" yaml:"port"`                              // 端口
	Node          int64  `mapstructure:"node" json:"node" yaml:"node"`                              // 节点
	ClusterName   string `mapstructure:"cluster-name" json:"clusterName" yaml:"cluster-name"`       // 集群名称
	Version       string `mapstructure:"version" json:"version" yaml:"version"`                     // 版本
	DbType        string `mapstructure:"db-type" json:"dbType" yaml:"db-type"`                      // 数据库类型:mysql(默认)|sqlite|sqlserver|postgresql
	ContextPath   string `mapstructure:"context-path" json:"contextPath" yaml:"context-path"`       // 请求上下文路径
	AesKey        string `mapstructure:"aes-key" json:"aesKey" yaml:"aes-key"`                      // AES Key
	SessionCron   string `mapstructure:"session-cron" json:"sessionCron" yaml:"session-cron"`       // 用户会话检查时间单位
	PrintRoute    bool   `mapstructure:"print-route" json:"printRoute" yaml:"print-route"`          // 打印路由日志
	EnableSwagger bool   `mapstructure:"enable-swagger" json:"enableSwagger" yaml:"enable-swagger"` // 启用 Swagger
	AllowOrigins  string `mapstructure:"allow-origins" json:"allowOrigins" yaml:"allow-origins"`    // 允许跨域配置
}
