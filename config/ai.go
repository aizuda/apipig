package config

// AI 定义 AI 网关运行时与凭据安全配置。
type AI struct {
	EncryptionKey      string `mapstructure:"encryption-key" json:"-" yaml:"encryption-key"`
	LogQueueSize       int    `mapstructure:"log-queue-size" json:"logQueueSize" yaml:"log-queue-size"`
	LogBatchSize       int    `mapstructure:"log-batch-size" json:"logBatchSize" yaml:"log-batch-size"`
	LogFlushIntervalMs int    `mapstructure:"log-flush-interval-ms" json:"logFlushIntervalMs" yaml:"log-flush-interval-ms"`
}
