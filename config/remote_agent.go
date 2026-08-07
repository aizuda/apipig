package config

// RemoteAgent defines controller-side liveness and dispatch settings.
type RemoteAgent struct {
	HeartbeatTimeoutSeconds   int `mapstructure:"heartbeat-timeout-seconds" json:"heartbeatTimeoutSeconds" yaml:"heartbeat-timeout-seconds"`
	CommandPollTimeoutSeconds int `mapstructure:"command-poll-timeout-seconds" json:"commandPollTimeoutSeconds" yaml:"command-poll-timeout-seconds"`
	DispatchLeaseSeconds      int `mapstructure:"dispatch-lease-seconds" json:"dispatchLeaseSeconds" yaml:"dispatch-lease-seconds"`
}
