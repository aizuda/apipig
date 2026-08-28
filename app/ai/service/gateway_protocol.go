package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zendev-sh/goai/provider"
)

type openAIChatRequest struct {
	Model               string          `json:"model"`
	Messages            []openAIMessage `json:"messages"`
	Tools               []openAITool    `json:"tools"`
	ToolChoice          json.RawMessage `json:"tool_choice"`
	MaxTokens           int             `json:"max_tokens"`
	MaxCompletionTokens int             `json:"max_completion_tokens"`
	Temperature         *float64        `json:"temperature"`
	TopP                *float64        `json:"top_p"`
	FrequencyPenalty    *float64        `json:"frequency_penalty"`
	PresencePenalty     *float64        `json:"presence_penalty"`
	Seed                *int            `json:"seed"`
	Stop                json.RawMessage `json:"stop"`
	Stream              bool            `json:"stream"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    json.RawMessage  `json:"content"`
	ToolCallID string           `json:"tool_call_id"`
	ToolCalls  []openAIToolCall `json:"tool_calls"`
}

type openAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type anthropicMessagesRequest struct {
	Model       string             `json:"model"`
	System      json.RawMessage    `json:"system"`
	Messages    []anthropicMessage `json:"messages"`
	Tools       []anthropicTool    `json:"tools"`
	ToolChoice  json.RawMessage    `json:"tool_choice"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature"`
	TopP        *float64           `json:"top_p"`
	TopK        *int               `json:"top_k"`
	Stop        []string           `json:"stop_sequences"`
	Stream      bool               `json:"stream"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

func parseOpenAIRequest(body []byte) (openAIChatRequest, provider.GenerateParams, error) {
	var req openAIChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return req, provider.GenerateParams{}, fmt.Errorf("请求体不是有效 JSON: %w", err)
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return req, provider.GenerateParams{}, errors.New("请求体缺少 model 字段")
	}
	if len(req.Model) > 200 {
		return req, provider.GenerateParams{}, errors.New("model 长度不能超过 200 个字符")
	}
	if len(req.Messages) == 0 || len(req.Messages) > maxGatewayMessages {
		return req, provider.GenerateParams{}, fmt.Errorf("messages 数量必须在 1 到 %d 之间", maxGatewayMessages)
	}
	if len(req.Tools) > maxGatewayTools {
		return req, provider.GenerateParams{}, fmt.Errorf("tools 数量不能超过 %d", maxGatewayTools)
	}
	params := provider.GenerateParams{
		MaxOutputTokens:  firstPositive(req.MaxCompletionTokens, req.MaxTokens),
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		FrequencyPenalty: req.FrequencyPenalty,
		PresencePenalty:  req.PresencePenalty,
		Seed:             req.Seed,
		StopSequences:    parseStopSequences(req.Stop),
		ToolChoice:       parseOpenAIToolChoice(req.ToolChoice),
	}
	toolNames := make(map[string]string)
	for _, message := range req.Messages {
		if strings.EqualFold(message.Role, "assistant") {
			for _, toolCall := range message.ToolCalls {
				toolNames[toolCall.ID] = toolCall.Function.Name
			}
		}
	}
	for _, message := range req.Messages {
		converted, system, err := convertOpenAIMessage(message, toolNames)
		if err != nil {
			return req, provider.GenerateParams{}, err
		}
		if system != "" {
			params.System = joinSystem(params.System, system)
			continue
		}
		params.Messages = append(params.Messages, converted)
	}
	for _, tool := range req.Tools {
		if tool.Type != "" && tool.Type != "function" {
			continue
		}
		if strings.TrimSpace(tool.Function.Name) == "" || len(tool.Function.Name) > 128 {
			return req, provider.GenerateParams{}, errors.New("工具名称不能为空且不能超过 128 个字符")
		}
		if len(tool.Function.Parameters) > 0 && !json.Valid(tool.Function.Parameters) {
			return req, provider.GenerateParams{}, fmt.Errorf("工具 %s 的 parameters 不是有效 JSON", tool.Function.Name)
		}
		params.Tools = append(params.Tools, provider.ToolDefinition{
			Name: tool.Function.Name, Description: tool.Function.Description, InputSchema: tool.Function.Parameters,
		})
	}
	return req, params, nil
}

func convertOpenAIMessage(message openAIMessage, toolNames map[string]string) (provider.Message, string, error) {
	role := strings.ToLower(strings.TrimSpace(message.Role))
	parts, err := parseOpenAIContent(message.Content)
	if err != nil {
		return provider.Message{}, "", err
	}
	switch role {
	case "system", "developer":
		return provider.Message{}, partsText(parts), nil
	case "user":
		return provider.Message{Role: provider.RoleUser, Content: parts}, "", nil
	case "assistant":
		for _, call := range message.ToolCalls {
			parts = append(parts, provider.Part{
				Type: provider.PartToolCall, ToolCallID: call.ID, ToolName: call.Function.Name, ToolInput: normalizeToolInput(call.Function.Arguments),
			})
		}
		return provider.Message{Role: provider.RoleAssistant, Content: parts}, "", nil
	case "tool":
		return provider.Message{Role: provider.RoleTool, Content: []provider.Part{{
			Type: provider.PartToolResult, ToolCallID: message.ToolCallID, ToolName: toolNames[message.ToolCallID], ToolOutput: partsText(parts),
		}}}, "", nil
	default:
		return provider.Message{}, "", fmt.Errorf("不支持的消息角色: %s", message.Role)
	}
}

func parseOpenAIContent(raw json.RawMessage) ([]provider.Part, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []provider.Part{{Type: provider.PartText, Text: text}}, nil
	}
	var blocks []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL struct {
			URL    string `json:"url"`
			Detail string `json:"detail"`
		} `json:"image_url"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, errors.New("message.content 必须是字符串或内容数组")
	}
	if len(blocks) > maxGatewayContentParts {
		return nil, fmt.Errorf("单条消息内容块不能超过 %d", maxGatewayContentParts)
	}
	parts := make([]provider.Part, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "text", "input_text":
			parts = append(parts, provider.Part{Type: provider.PartText, Text: block.Text})
		case "image_url":
			parts = append(parts, provider.Part{Type: provider.PartImage, URL: block.ImageURL.URL, Detail: block.ImageURL.Detail})
		}
	}
	return parts, nil
}

// isAudioChatRequest identifies OpenAI-compatible multimodal audio requests.
// It intentionally only checks the message content type; the raw proxy path
// is responsible for validating the model and forwarding provider-specific
// fields such as asr_options unchanged.
func isAudioChatRequest(body []byte) bool {
	var request struct {
		Messages []struct {
			Content []struct {
				Type string `json:"type"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return false
	}
	for _, message := range request.Messages {
		for _, part := range message.Content {
			if strings.EqualFold(strings.TrimSpace(part.Type), "input_audio") {
				return true
			}
		}
	}
	return false
}

func parseAnthropicRequest(body []byte) (anthropicMessagesRequest, provider.GenerateParams, error) {
	var req anthropicMessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return req, provider.GenerateParams{}, fmt.Errorf("请求体不是有效 JSON: %w", err)
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return req, provider.GenerateParams{}, errors.New("请求体缺少 model 字段")
	}
	if len(req.Model) > 200 {
		return req, provider.GenerateParams{}, errors.New("model 长度不能超过 200 个字符")
	}
	if len(req.Messages) == 0 || len(req.Messages) > maxGatewayMessages {
		return req, provider.GenerateParams{}, fmt.Errorf("messages 数量必须在 1 到 %d 之间", maxGatewayMessages)
	}
	if len(req.Tools) > maxGatewayTools {
		return req, provider.GenerateParams{}, fmt.Errorf("tools 数量不能超过 %d", maxGatewayTools)
	}
	if req.MaxTokens <= 0 {
		return req, provider.GenerateParams{}, errors.New("max_tokens 必须大于 0")
	}
	params := provider.GenerateParams{
		System:          parseAnthropicSystem(req.System),
		MaxOutputTokens: req.MaxTokens,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		TopK:            req.TopK,
		StopSequences:   req.Stop,
		ToolChoice:      parseAnthropicToolChoice(req.ToolChoice),
	}
	for _, message := range req.Messages {
		converted, err := convertAnthropicMessage(message)
		if err != nil {
			return req, provider.GenerateParams{}, err
		}
		params.Messages = append(params.Messages, converted)
	}
	for _, tool := range req.Tools {
		if strings.TrimSpace(tool.Name) == "" || len(tool.Name) > 128 {
			return req, provider.GenerateParams{}, errors.New("工具名称不能为空且不能超过 128 个字符")
		}
		if len(tool.InputSchema) > 0 && !json.Valid(tool.InputSchema) {
			return req, provider.GenerateParams{}, fmt.Errorf("工具 %s 的 input_schema 不是有效 JSON", tool.Name)
		}
		params.Tools = append(params.Tools, provider.ToolDefinition{
			Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema,
		})
	}
	return req, params, nil
}

func convertAnthropicMessage(message anthropicMessage) (provider.Message, error) {
	role := provider.RoleUser
	if strings.EqualFold(message.Role, "assistant") {
		role = provider.RoleAssistant
	}
	var text string
	if err := json.Unmarshal(message.Content, &text); err == nil {
		return provider.Message{Role: role, Content: []provider.Part{{Type: provider.PartText, Text: text}}}, nil
	}
	var blocks []struct {
		Type      string          `json:"type"`
		Text      string          `json:"text"`
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		Input     json.RawMessage `json:"input"`
		ToolUseID string          `json:"tool_use_id"`
		Content   json.RawMessage `json:"content"`
		Source    struct {
			Type      string `json:"type"`
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
			URL       string `json:"url"`
		} `json:"source"`
	}
	if err := json.Unmarshal(message.Content, &blocks); err != nil {
		return provider.Message{}, errors.New("messages.content 必须是字符串或内容数组")
	}
	if len(blocks) > maxGatewayContentParts {
		return provider.Message{}, fmt.Errorf("单条消息内容块不能超过 %d", maxGatewayContentParts)
	}
	parts := make([]provider.Part, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "text":
			parts = append(parts, provider.Part{Type: provider.PartText, Text: block.Text})
		case "image":
			imageURL := block.Source.URL
			if block.Source.Type == "base64" {
				imageURL = "data:" + block.Source.MediaType + ";base64," + block.Source.Data
			}
			parts = append(parts, provider.Part{Type: provider.PartImage, URL: imageURL, MediaType: block.Source.MediaType})
		case "tool_use":
			parts = append(parts, provider.Part{Type: provider.PartToolCall, ToolCallID: block.ID, ToolName: block.Name, ToolInput: normalizeJSON(block.Input)})
		case "tool_result":
			parts = append(parts, provider.Part{Type: provider.PartToolResult, ToolCallID: block.ToolUseID, ToolOutput: rawContentText(block.Content)})
		}
	}
	if len(parts) > 0 && parts[0].Type == provider.PartToolResult {
		role = provider.RoleTool
	}
	return provider.Message{Role: role, Content: parts}, nil
}

func buildOpenAIResponse(model string, result *provider.GenerateResult) map[string]any {
	message := map[string]any{"role": "assistant", "content": result.Text}
	if len(result.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, 0, len(result.ToolCalls))
		for _, call := range result.ToolCalls {
			toolCalls = append(toolCalls, map[string]any{
				"id": call.ID, "type": "function", "function": map[string]any{"name": call.Name, "arguments": string(call.Input)},
			})
		}
		message["tool_calls"] = toolCalls
		if result.Text == "" {
			message["content"] = nil
		}
	}
	return map[string]any{
		"id": responseID(result.Response.ID, "chatcmpl"), "object": "chat.completion", "created": time.Now().Unix(),
		"model":   responseModel(model, result.Response.Model),
		"choices": []map[string]any{{"index": 0, "message": message, "finish_reason": openAIFinishReason(result.FinishReason)}},
		"usage":   openAIUsage(result.Usage),
	}
}

func buildAnthropicResponse(model string, result *provider.GenerateResult) map[string]any {
	content := make([]map[string]any, 0, len(result.ToolCalls)+1)
	if result.Text != "" {
		content = append(content, map[string]any{"type": "text", "text": result.Text})
	}
	for _, call := range result.ToolCalls {
		var input any = map[string]any{}
		_ = json.Unmarshal(call.Input, &input)
		content = append(content, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": input})
	}
	return map[string]any{
		"id": responseID(result.Response.ID, "msg"), "type": "message", "role": "assistant",
		"model": responseModel(model, result.Response.Model), "content": content,
		"stop_reason": anthropicStopReason(result.FinishReason), "stop_sequence": nil,
		"usage": anthropicUsage(result.Usage),
	}
}

func openAIUsage(usage provider.Usage) map[string]any {
	promptTokens := usage.InputTokens + usage.CacheReadTokens + usage.CacheWriteTokens
	total := usage.TotalTokens
	if total == 0 {
		total = promptTokens + usage.OutputTokens
	}
	result := map[string]any{"prompt_tokens": promptTokens, "completion_tokens": usage.OutputTokens, "total_tokens": total}
	if usage.CacheReadTokens > 0 {
		result["prompt_tokens_details"] = map[string]any{"cached_tokens": usage.CacheReadTokens}
	}
	if usage.ReasoningTokens > 0 {
		result["completion_tokens_details"] = map[string]any{"reasoning_tokens": usage.ReasoningTokens}
	}
	return result
}

func anthropicUsage(usage provider.Usage) map[string]any {
	return map[string]any{
		"input_tokens":                usage.InputTokens,
		"output_tokens":               usage.OutputTokens,
		"cache_read_input_tokens":     usage.CacheReadTokens,
		"cache_creation_input_tokens": usage.CacheWriteTokens,
	}
}

func openAIFinishReason(reason provider.FinishReason) string {
	switch reason {
	case provider.FinishToolCalls:
		return "tool_calls"
	case provider.FinishLength:
		return "length"
	case provider.FinishContentFilter:
		return "content_filter"
	default:
		return "stop"
	}
}

func anthropicStopReason(reason provider.FinishReason) string {
	switch reason {
	case provider.FinishToolCalls:
		return "tool_use"
	case provider.FinishLength:
		return "max_tokens"
	default:
		return "end_turn"
	}
}

func parseStopSequences(raw json.RawMessage) []string {
	var items []string
	if json.Unmarshal(raw, &items) == nil {
		return items
	}
	var item string
	if json.Unmarshal(raw, &item) == nil && item != "" {
		return []string{item}
	}
	return nil
}

func parseOpenAIToolChoice(raw json.RawMessage) string {
	var choice string
	if json.Unmarshal(raw, &choice) == nil {
		return choice
	}
	var named struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &named) == nil {
		return named.Function.Name
	}
	return ""
}

func parseAnthropicToolChoice(raw json.RawMessage) string {
	var choice struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &choice) != nil {
		return ""
	}
	switch choice.Type {
	case "any":
		return "required"
	case "tool":
		return choice.Name
	default:
		return choice.Type
	}
}

func parseAnthropicSystem(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	items := make([]string, 0, len(blocks))
	for _, block := range blocks {
		items = append(items, block.Text)
	}
	return strings.Join(items, "\n")
}

func rawContentText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		items := make([]string, 0, len(blocks))
		for _, block := range blocks {
			if block.Type == "text" && block.Text != "" {
				items = append(items, block.Text)
			}
		}
		return strings.Join(items, "\n")
	}
	return string(raw)
}

func partsText(parts []provider.Part) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Type == provider.PartText && part.Text != "" {
			items = append(items, part.Text)
		}
	}
	return strings.Join(items, "\n")
}

func normalizeJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return raw
}

func normalizeToolInput(raw json.RawMessage) json.RawMessage {
	var encoded string
	if json.Unmarshal(raw, &encoded) == nil && json.Valid([]byte(encoded)) {
		return json.RawMessage(encoded)
	}
	return normalizeJSON(raw)
}

func joinSystem(current, next string) string {
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return current + "\n" + next
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func responseID(value, prefix string) string {
	if value != "" {
		return value
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func responseModel(requested, actual string) string {
	if actual != "" {
		return actual
	}
	return requested
}
