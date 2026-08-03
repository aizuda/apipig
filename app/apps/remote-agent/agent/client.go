package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	remoteReq "apipig/app/apps/remote-agent/model/request"
	remoteResp "apipig/app/apps/remote-agent/model/response"
)

type RegistrationInfo = remoteReq.RegisterRequest
type TaskResult = remoteReq.TaskResultRequest
type Command = remoteResp.CommandDispatch

type APIError struct {
	Message string
}

func (e *APIError) Error() string { return e.Message }

type Client struct {
	baseURL           string
	registrationToken string
	httpClient        *http.Client
	mu                sync.RWMutex
	agentToken        string
}

func NewClient(config Config) *Client {
	return &Client{
		baseURL:           strings.TrimRight(config.ControllerURL, "/") + "/v1/remote-agent",
		registrationToken: config.RegistrationToken,
		httpClient:        &http.Client{Timeout: time.Duration(config.RequestTimeoutSeconds) * time.Second},
	}
}

func (c *Client) Register(ctx context.Context, info RegistrationInfo) (remoteResp.RegisterResult, error) {
	var result remoteResp.RegisterResult
	err := c.do(ctx, http.MethodPost, "/register", info, map[string]string{
		"X-Agent-Registration-Token": c.registrationToken,
	}, &result)
	if err == nil {
		c.mu.Lock()
		c.agentToken = result.AgentToken
		c.mu.Unlock()
	}
	return result, err
}

func (c *Client) Heartbeat(ctx context.Context, heartbeat remoteReq.HeartbeatRequest) error {
	return c.doAgent(ctx, http.MethodPost, "/heartbeat", heartbeat, nil)
}

func (c *Client) NextCommand(ctx context.Context, waitSeconds int) (*Command, error) {
	var command *Command
	path := "/command/next?waitSeconds=" + url.QueryEscape(strconv.Itoa(waitSeconds))
	err := c.doAgent(ctx, http.MethodGet, path, nil, &command)
	return command, err
}

func (c *Client) Acknowledge(ctx context.Context, commandID fmt.Stringer) error {
	return c.doAgent(ctx, http.MethodPost, "/command/acknowledge", map[string]string{
		"commandId": commandID.String(),
	}, nil)
}

func (c *Client) Complete(ctx context.Context, result TaskResult) error {
	return c.doAgent(ctx, http.MethodPost, "/task/result", result, nil)
}

func (c *Client) UploadLogs(ctx context.Context, request remoteReq.TaskLogUploadRequest) error {
	return c.doAgent(ctx, http.MethodPost, "/task/logs", request, nil)
}

func (c *Client) doAgent(ctx context.Context, method, path string, body, result any) error {
	c.mu.RLock()
	token := c.agentToken
	c.mu.RUnlock()
	if token == "" {
		return &APIError{Message: "agent is not registered"}
	}
	return c.do(ctx, method, path, body, map[string]string{"Authorization": "Bearer " + token}, result)
}

func (c *Client) do(ctx context.Context, method, path string, body any, headers map[string]string, result any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var envelope struct {
		Code string          `json:"code"`
		Data json.RawMessage `json:"data"`
		Msg  string          `json:"msg"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&envelope); err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != "1" {
		message := strings.TrimSpace(envelope.Msg)
		if message == "" {
			message = response.Status
		}
		return &APIError{Message: message}
	}
	if result == nil || len(envelope.Data) == 0 || bytes.Equal(envelope.Data, []byte("null")) {
		return nil
	}
	return json.Unmarshal(envelope.Data, result)
}
