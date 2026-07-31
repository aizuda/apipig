package config

// CodeReview 定义 Git WebHook 代码评审的队列、Git 和 AI 超时配置。
type CodeReview struct {
	WorkerCount     int    `mapstructure:"worker-count" json:"workerCount" yaml:"worker-count"`
	QueueSize       int    `mapstructure:"queue-size" json:"queueSize" yaml:"queue-size"`
	WorkspaceRoot   string `mapstructure:"workspace-root" json:"workspaceRoot" yaml:"workspace-root"`
	GitTimeoutSec   int    `mapstructure:"git-timeout-sec" json:"gitTimeoutSec" yaml:"git-timeout-sec"`
	AITimeoutSec    int    `mapstructure:"ai-timeout-sec" json:"aiTimeoutSec" yaml:"ai-timeout-sec"`
	MaxDiffBytes    int    `mapstructure:"max-diff-bytes" json:"maxDiffBytes" yaml:"max-diff-bytes"`
	MaxChangedFiles int    `mapstructure:"max-changed-files" json:"maxChangedFiles" yaml:"max-changed-files"`
}
