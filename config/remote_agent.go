package config

// RemoteAgent defines Controller-side registration and liveness settings.
type RemoteAgent struct {
	RegistrationToken         string `mapstructure:"registration-token" json:"-" yaml:"registration-token"`
	HeartbeatTimeoutSeconds   int    `mapstructure:"heartbeat-timeout-seconds" json:"heartbeatTimeoutSeconds" yaml:"heartbeat-timeout-seconds"`
	OfflineCheckCron          string `mapstructure:"offline-check-cron" json:"offlineCheckCron" yaml:"offline-check-cron"`
	CommandPollTimeoutSeconds int    `mapstructure:"command-poll-timeout-seconds" json:"commandPollTimeoutSeconds" yaml:"command-poll-timeout-seconds"`
	DispatchLeaseSeconds      int    `mapstructure:"dispatch-lease-seconds" json:"dispatchLeaseSeconds" yaml:"dispatch-lease-seconds"`
}
