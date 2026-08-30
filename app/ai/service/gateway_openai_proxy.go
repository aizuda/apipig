package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	aiReq "apipig/app/ai/model/request"

	"github.com/gofiber/fiber/v2"
)

const maxGatewayModelIDBytes = 200

// Embeddings provides OpenAI-compatible POST /v1/embeddings passthrough.
func (s *GatewayService) Embeddings(c *fiber.Ctx) error {
	return s.proxyStandardOpenAI(c, "/embeddings", maxGatewayRequestBodyBytes)
}

// ImageGenerations provides OpenAI-compatible POST /v1/images/generations passthrough.
func (s *GatewayService) ImageGenerations(c *fiber.Ctx) error {
	return s.proxyStandardOpenAI(c, "/images/generations", maxGatewayRequestBodyBytes)
}

// Rerank provides the commonly used OpenAI-compatible POST /v1/rerank extension.
func (s *GatewayService) Rerank(c *fiber.Ctx) error {
	return s.proxyStandardOpenAI(c, "/rerank", maxGatewayRequestBodyBytes)
}

// AudioSpeech provides OpenAI-compatible POST /v1/audio/speech passthrough.
func (s *GatewayService) AudioSpeech(c *fiber.Ctx) error {
	return s.proxyStandardOpenAI(c, "/audio/speech", maxGatewayRequestBodyBytes)
}

// AudioTranscriptions provides OpenAI-compatible multipart transcription passthrough.
func (s *GatewayService) AudioTranscriptions(c *fiber.Ctx) error {
	if err := validateAudioTranscriptionRequest(c.Body(), c.Get(fiber.HeaderContentType)); err != nil {
		status := fiber.StatusBadRequest
		if errors.Is(err, errUnsupportedMultipartMediaType) {
			status = fiber.StatusUnsupportedMediaType
		}
		if errors.Is(err, errAudioFileTooLarge) {
			status = fiber.StatusRequestEntityTooLarge
		}
		return writeGatewayError(c, "openai", status, err)
	}
	return s.proxyStandardOpenAI(c, "/audio/transcriptions", maxAudioTranscriptionRequestBodyBytes)
}

func (s *GatewayService) proxyStandardOpenAI(c *fiber.Ctx, upstream string, maxBodyBytes int) error {
	err := s.ProxyOpenAI(&aiReq.GatewayProxyParams{
		Ctx: c, RawBody: c.Body(), Upstream: upstream, MaxBodyBytes: maxBodyBytes,
	})
	if err == nil {
		return nil
	}
	status := fiber.StatusInternalServerError
	if fiberErr, ok := errors.AsType[*fiber.Error](err); ok {
		status = fiberErr.Code
	}
	return writeGatewayError(c, "openai", status, err)
}

func parseRequestModelByContentType(body []byte, contentType string) (string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		return parseMultipartRequestModel(body, params["boundary"])
	}
	if err != nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/") {
		return "", errors.New("multipart/form-data Content-Type 无效")
	}
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", errors.New("请求体不是有效的 JSON")
	}
	return validateRequestModel(payload.Model)
}

func replaceRequestModelByContentType(body []byte, contentType, modelName string) ([]byte, string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		return replaceMultipartRequestModel(body, params["boundary"], modelName)
	}
	replaced, err := replaceRequestModel(body, modelName)
	return replaced, contentType, err
}

func parseMultipartRequestModel(body []byte, boundary string) (string, error) {
	if boundary == "" {
		return "", errors.New("multipart/form-data 缺少 boundary")
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return "", errors.New("请求体缺少 model 字段")
		}
		if err != nil {
			return "", errors.New("multipart/form-data 请求体无效")
		}
		if part.FormName() != "model" {
			_ = part.Close()
			continue
		}
		value, err := io.ReadAll(io.LimitReader(part, maxGatewayModelIDBytes+1))
		_ = part.Close()
		if err != nil {
			return "", errors.New("读取 model 字段失败")
		}
		return validateRequestModel(string(value))
	}
}

func replaceMultipartRequestModel(body []byte, boundary, modelName string) ([]byte, string, error) {
	if boundary == "" {
		return nil, "", errors.New("multipart/form-data 缺少 boundary")
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var output bytes.Buffer
	writer := multipart.NewWriter(&output)
	replaced := false
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", errors.New("multipart/form-data 请求体无效")
		}
		headers := cloneMIMEHeader(part.Header)
		destination, err := writer.CreatePart(headers)
		if err != nil {
			return nil, "", err
		}
		if part.FormName() == "model" {
			_, err = io.WriteString(destination, modelName)
			replaced = true
		} else {
			_, err = io.Copy(destination, part)
		}
		_ = part.Close()
		if err != nil {
			return nil, "", err
		}
	}
	if !replaced {
		return nil, "", errors.New("请求体缺少 model 字段")
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return output.Bytes(), writer.FormDataContentType(), nil
}

func cloneMIMEHeader(source textproto.MIMEHeader) textproto.MIMEHeader {
	result := make(textproto.MIMEHeader, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}

var (
	errUnsupportedMultipartMediaType = errors.New("语音识别请求必须使用 multipart/form-data")
	errAudioFileTooLarge             = errors.New("语音识别文件不能超过 25 MiB")
)

func validateAudioTranscriptionRequest(body []byte, contentType string) error {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") || params["boundary"] == "" {
		return errUnsupportedMultipartMediaType
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	foundFile := false
	foundModel := false
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.New("multipart/form-data 请求体无效")
		}
		switch part.FormName() {
		case "file":
			foundFile = true
			size, readErr := io.Copy(io.Discard, io.LimitReader(part, maxAudioTranscriptionFileBytes+1))
			if readErr != nil {
				_ = part.Close()
				return errors.New("读取语音识别文件失败")
			}
			if size > maxAudioTranscriptionFileBytes {
				_ = part.Close()
				return errAudioFileTooLarge
			}
		case "model":
			foundModel = true
		}
		_ = part.Close()
	}
	if !foundFile {
		return errors.New("请求体缺少 file 字段")
	}
	if !foundModel {
		return errors.New("请求体缺少 model 字段")
	}
	return nil
}

func validateRequestModel(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("请求体缺少 model 字段")
	}
	if len([]byte(value)) > maxGatewayModelIDBytes {
		return "", fmt.Errorf("model 不能超过 %d 字节", maxGatewayModelIDBytes)
	}
	return value, nil
}
