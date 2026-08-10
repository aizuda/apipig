package agent

import (
	buildversion "apipig/version"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ControllerURL         string   `yaml:"controller-url"`
	RegistrationToken     string   `yaml:"registration-token"`
	AgentKey              string   `yaml:"agent-key"`
	Name                  string   `yaml:"name"`
	WorkspaceRoot         string   `yaml:"workspace-root"`
	CodexCommand          string   `yaml:"codex-command"`
	CodexArgs             []string `yaml:"codex-args"`
	ClaudeCommand         string   `yaml:"claude-command"`
	ClaudeArgs            []string `yaml:"claude-args"`
	PollWaitSeconds       int      `yaml:"poll-wait-seconds"`
	RequestTimeoutSeconds int      `yaml:"request-timeout-seconds"`
	LogFile               string   `yaml:"log-file"`
	hostname              string
	sourcePath            string
}

func LoadConfig(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.UnmarshalStrict(content, &config); err != nil {
		return Config{}, err
	}
	if err := config.normalize(); err != nil {
		return Config{}, err
	}
	config.sourcePath, err = filepath.Abs(path)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c *Config) normalize() error {
	c.ControllerURL = strings.TrimRight(strings.TrimSpace(c.ControllerURL), "/")
	c.RegistrationToken = strings.TrimSpace(c.RegistrationToken)
	c.AgentKey = strings.TrimSpace(c.AgentKey)
	c.Name = strings.TrimSpace(c.Name)
	c.WorkspaceRoot = strings.TrimSpace(c.WorkspaceRoot)
	c.CodexCommand = strings.TrimSpace(c.CodexCommand)
	c.ClaudeCommand = strings.TrimSpace(c.ClaudeCommand)
	c.LogFile = strings.TrimSpace(c.LogFile)
	controllerURL, err := url.ParseRequestURI(c.ControllerURL)
	if err != nil || (controllerURL.Scheme != "http" && controllerURL.Scheme != "https") || controllerURL.Host == "" {
		return errors.New("controller-url must be an HTTP or HTTPS URL")
	}
	if c.RegistrationToken == "" {
		return errors.New("registration-token is required")
	}
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	c.hostname = strings.TrimSpace(hostname)
	if c.hostname == "" {
		return errors.New("hostname cannot be empty")
	}
	if c.AgentKey == "" {
		c.AgentKey = c.hostname
	}
	if c.Name == "" {
		c.Name = c.hostname
	}
	if c.AgentKey == "" || c.Name == "" {
		return errors.New("agent-key and name cannot be empty")
	}
	if c.WorkspaceRoot == "" {
		c.WorkspaceRoot = "."
	}
	c.WorkspaceRoot, err = filepath.Abs(c.WorkspaceRoot)
	if err != nil {
		return err
	}
	workspaceInfo, err := os.Stat(c.WorkspaceRoot)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(c.WorkspaceRoot, 0750); err != nil {
			return errors.New("create workspace-root: " + err.Error())
		}
		workspaceInfo, err = os.Stat(c.WorkspaceRoot)
	}
	if err != nil {
		return errors.New("inspect workspace-root: " + err.Error())
	}
	if !workspaceInfo.IsDir() {
		return errors.New("workspace-root must reference a directory")
	}
	if c.CodexCommand == "" {
		c.CodexCommand = "codex"
	}
	if len(c.CodexArgs) == 0 {
		c.CodexArgs = []string{
			"exec", "--json", "--skip-git-repo-check", "-",
		}
	}
	if c.ClaudeCommand == "" {
		c.ClaudeCommand = "claude"
	}
	if len(c.ClaudeArgs) == 0 {
		c.ClaudeArgs = []string{"-p"}
	}
	if c.PollWaitSeconds <= 0 || c.PollWaitSeconds > 25 {
		c.PollWaitSeconds = 25
	}
	if c.RequestTimeoutSeconds <= c.PollWaitSeconds {
		c.RequestTimeoutSeconds = c.PollWaitSeconds + 10
	}
	if c.LogFile == "" {
		c.LogFile = "remote-agent.log"
	}
	if !filepath.IsAbs(c.LogFile) {
		c.LogFile, err = filepath.Abs(c.LogFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c Config) Registration() RegistrationInfo {
	return RegistrationInfo{
		AgentKey: c.AgentKey, Hostname: c.hostname, OperatingSystem: runtime.GOOS,
		Architecture: runtime.GOARCH, CPUInfo: logicalCPUInfo(), AgentVersion: buildversion.Version,
	}
}
