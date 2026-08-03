package config

type Server struct {
	// AI 网关运行时与安全配置
	AI AI `mapstructure:"ai" json:"ai" yaml:"ai"`

	// CodeReview 定义 Git WebHook 代码评审任务配置。
	CodeReview CodeReview `mapstructure:"code-review" json:"codeReview" yaml:"code-review"`

	// RemoteAgent defines the Remote Agent Controller settings.
	RemoteAgent RemoteAgent `mapstructure:"remote-agent" json:"remoteAgent" yaml:"remote-agent"`

	// 授权认证相关配置
	JWT JWT `mapstructure:"jwt" json:"jwt" yaml:"jwt"`

	// 系统相关配置
	System System `mapstructure:"system" json:"system" yaml:"system"`

	// 设备相关配置
	Device Device `mapstructure:"device" json:"device" yaml:"device"`

	// 关系型数据库
	Database Database `mapstructure:"database" json:"database" yaml:"database"`

	// 时序数据库
	TimeSeriesDb TimeSeriesDb `mapstructure:"time-series-db" json:"timeSeriesDb" yaml:"time-series-db"`

	// 上传文件
	Upload Upload `mapstructure:"upload" json:"upload" yaml:"upload"`

	// 插件配置
	Plugin Plugin `mapstructure:"plugin" json:"plugin" yaml:"plugin"`
}
