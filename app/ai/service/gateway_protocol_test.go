package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zendev-sh/goai/provider"
)

func TestParseOpenAIRequest(t *testing.T) {
	body := []byte(`{
		"model":"gpt-test",
		"messages":[
			{"role":"system","content":"be concise"},
			{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"https://example.com/a.png","detail":"low"}}]},
			{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Shanghai\"}"}}]},
			{"role":"tool","tool_call_id":"call-1","content":"sunny"}
		],
		"tools":[{"type":"function","function":{"name":"weather","description":"weather lookup","parameters":{"type":"object"}}}],
		"tool_choice":{"type":"function","function":{"name":"weather"}},
		"max_completion_tokens":128,
		"stream":true
	}`)

	req, params, err := parseOpenAIRequest(body)
	require.NoError(t, err)
	assert.Equal(t, "gpt-test", req.Model)
	assert.True(t, req.Stream)
	assert.Equal(t, "be concise", params.System)
	assert.Equal(t, 128, params.MaxOutputTokens)
	assert.Equal(t, "weather", params.ToolChoice)
	require.Len(t, params.Messages, 3)
	assert.Equal(t, provider.RoleUser, params.Messages[0].Role)
	assert.Equal(t, provider.PartImage, params.Messages[0].Content[1].Type)
	assert.Equal(t, provider.PartToolCall, params.Messages[1].Content[0].Type)
	assert.JSONEq(t, `{"city":"Shanghai"}`, string(params.Messages[1].Content[0].ToolInput))
	assert.Equal(t, provider.RoleTool, params.Messages[2].Role)
	assert.Equal(t, "weather", params.Messages[2].Content[0].ToolName)
	require.Len(t, params.Tools, 1)
	assert.Equal(t, "weather", params.Tools[0].Name)
}

func TestParseAnthropicRequest(t *testing.T) {
	body := []byte(`{
		"model":"claude-test",
		"system":[{"type":"text","text":"first"},{"type":"text","text":"second"}],
		"messages":[
			{"role":"user","content":"hello"},
			{"role":"assistant","content":[{"type":"tool_use","id":"tool-1","name":"lookup","input":{"id":1}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1","content":"done"}]}
		],
		"tools":[{"name":"lookup","description":"lookup item","input_schema":{"type":"object"}}],
		"tool_choice":{"type":"any"},
		"max_tokens":256
	}`)

	req, params, err := parseAnthropicRequest(body)
	require.NoError(t, err)
	assert.Equal(t, "claude-test", req.Model)
	assert.Equal(t, "first\nsecond", params.System)
	assert.Equal(t, 256, params.MaxOutputTokens)
	assert.Equal(t, "required", params.ToolChoice)
	require.Len(t, params.Messages, 3)
	assert.Equal(t, provider.PartToolCall, params.Messages[1].Content[0].Type)
	assert.Equal(t, provider.RoleTool, params.Messages[2].Role)
	assert.Equal(t, "done", params.Messages[2].Content[0].ToolOutput)
	require.Len(t, params.Tools, 1)
	assert.Equal(t, "lookup", params.Tools[0].Name)
}

func TestProtocolFinishReasonMapping(t *testing.T) {
	assert.Equal(t, "tool_calls", openAIFinishReason(provider.FinishToolCalls))
	assert.Equal(t, "max_tokens", anthropicStopReason(provider.FinishLength))
}

func TestProtocolRequestValidation(t *testing.T) {
	_, _, err := parseOpenAIRequest([]byte(`{"model":"gpt-test","messages":[]}`))
	assert.ErrorContains(t, err, "messages 数量")

	_, _, err = parseAnthropicRequest([]byte(`{"model":"claude-test","messages":[{"role":"user","content":"hello"}],"max_tokens":0}`))
	assert.ErrorContains(t, err, "max_tokens")
}
