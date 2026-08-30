package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

const maxQwenASRFormFieldBytes = 64 << 10

type qwenASRMultipartRequest struct {
	Model          string
	Audio          []byte
	AudioMediaType string
	Language       string
	EnableITN      *bool
	ASROptions     map[string]any
	ResponseFormat string
}

func isQwenTranscriptionTarget(target routeTarget, upstream string) bool {
	if upstream != "/audio/transcriptions" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(target.Provider.Protocol)) {
	case "qwen", "dashscope":
		return true
	default:
		return false
	}
}

func adaptQwenTranscriptionRequest(body []byte, contentType string) (string, []byte, string, string, error) {
	request, err := parseQwenASRMultipartRequest(body, contentType)
	if err != nil {
		return "", nil, "", "", err
	}
	responseFormat := strings.ToLower(strings.TrimSpace(request.ResponseFormat))
	if responseFormat == "" {
		responseFormat = "json"
	}
	if responseFormat != "json" && responseFormat != "text" {
		return "", nil, "", "", fmt.Errorf("Qwen ASR 不支持 response_format=%s，仅支持 json 或 text", responseFormat)
	}
	options := request.ASROptions
	if options == nil {
		options = make(map[string]any)
	}
	if request.Language != "" {
		options["language"] = request.Language
	}
	if request.EnableITN != nil {
		options["enable_itn"] = *request.EnableITN
	}
	audioData := "data:" + request.AudioMediaType + ";base64," + base64.StdEncoding.EncodeToString(request.Audio)
	payload := map[string]any{
		"model": request.Model,
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{{
				"type":        "input_audio",
				"input_audio": map[string]any{"data": audioData},
			}},
		}},
		"stream": false,
	}
	if len(options) > 0 {
		payload["asr_options"] = options
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", nil, "", "", err
	}
	return "/chat/completions", encoded, "application/json", responseFormat, nil
}

func parseQwenASRMultipartRequest(body []byte, contentType string) (qwenASRMultipartRequest, error) {
	var result qwenASRMultipartRequest
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") || params["boundary"] == "" {
		return result, errUnsupportedMultipartMediaType
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return result, errors.New("multipart/form-data 请求体无效")
		}
		name := part.FormName()
		if name == "file" {
			result.Audio, err = io.ReadAll(io.LimitReader(part, maxAudioTranscriptionFileBytes+1))
			if err == nil && len(result.Audio) > maxAudioTranscriptionFileBytes {
				err = errAudioFileTooLarge
			}
			result.AudioMediaType = qwenASRAudioMediaType(part, result.Audio)
			_ = part.Close()
			if err != nil {
				return result, err
			}
			continue
		}
		value, readErr := io.ReadAll(io.LimitReader(part, maxQwenASRFormFieldBytes+1))
		_ = part.Close()
		if readErr != nil {
			return result, readErr
		}
		if len(value) > maxQwenASRFormFieldBytes {
			return result, fmt.Errorf("表单字段 %s 过长", name)
		}
		switch name {
		case "model":
			result.Model = strings.TrimSpace(string(value))
		case "language":
			result.Language = strings.TrimSpace(string(value))
		case "enable_itn":
			parsed, parseErr := strconv.ParseBool(strings.TrimSpace(string(value)))
			if parseErr != nil {
				return result, errors.New("enable_itn 必须是布尔值")
			}
			result.EnableITN = &parsed
		case "asr_options":
			if err := json.Unmarshal(value, &result.ASROptions); err != nil {
				return result, errors.New("asr_options 必须是 JSON 对象")
			}
		case "response_format":
			result.ResponseFormat = strings.TrimSpace(string(value))
		case "stream":
			stream, parseErr := strconv.ParseBool(strings.TrimSpace(string(value)))
			if parseErr != nil {
				return result, errors.New("stream 必须是布尔值")
			}
			if stream {
				return result, errors.New("Qwen ASR 暂不支持通过 audio/transcriptions 流式返回")
			}
		}
	}
	if result.Model == "" {
		return result, errors.New("请求体缺少 model 字段")
	}
	if len(result.Audio) == 0 {
		return result, errors.New("请求体缺少有效的 file 字段")
	}
	return result, nil
}

func qwenASRAudioMediaType(part *multipart.Part, data []byte) string {
	if value := strings.TrimSpace(part.Header.Get("Content-Type")); value != "" && value != "application/octet-stream" {
		if mediaType, _, err := mime.ParseMediaType(value); err == nil {
			return mediaType
		}
	}
	if value := mime.TypeByExtension(strings.ToLower(filepath.Ext(part.FileName()))); value != "" {
		if mediaType, _, err := mime.ParseMediaType(value); err == nil {
			return mediaType
		}
	}
	return http.DetectContentType(data)
}

func adaptQwenTranscriptionResponse(body []byte, headers http.Header, responseFormat string) ([]byte, http.Header, error) {
	var payload struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || len(payload.Choices) == 0 {
		return nil, headers, errors.New("Qwen ASR 返回了无效的聊天响应")
	}
	text, err := qwenASRResponseText(payload.Choices[0].Message.Content)
	if err != nil {
		return nil, headers, err
	}
	resultHeaders := headers.Clone()
	resultHeaders.Del("Content-Length")
	if responseFormat == "text" {
		resultHeaders.Set("Content-Type", "text/plain; charset=utf-8")
		return []byte(text), resultHeaders, nil
	}
	result := map[string]any{"text": text}
	if len(payload.Usage) > 0 && string(payload.Usage) != "null" {
		result["usage"] = payload.Usage
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, headers, err
	}
	resultHeaders.Set("Content-Type", "application/json")
	return encoded, resultHeaders, nil
}

func qwenASRResponseText(content json.RawMessage) (string, error) {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return text, nil
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) != nil {
		return "", errors.New("Qwen ASR 响应缺少文本内容")
	}
	items := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			items = append(items, block.Text)
		}
	}
	if len(items) == 0 {
		return "", errors.New("Qwen ASR 响应缺少文本内容")
	}
	return strings.Join(items, ""), nil
}
