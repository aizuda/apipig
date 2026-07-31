package service

import (
	"math"
	"testing"

	aiReq "apipig/app/ai/model/request"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAIChatParams(t *testing.T) {
	validTemperature := 0.7
	require.NoError(t, validateAIChatParams(&aiReq.AIChatParams{
		Temperature: &validTemperature,
		MaxTokens:   2048,
	}))

	tests := []struct {
		name        string
		temperature float64
		maxTokens   int
		errorText   string
	}{
		{name: "temperature below minimum", temperature: -0.1, maxTokens: 1, errorText: "temperature 必须在 0 到 2 之间"},
		{name: "temperature above maximum", temperature: 2.1, maxTokens: 1, errorText: "temperature 必须在 0 到 2 之间"},
		{name: "temperature is NaN", temperature: math.NaN(), maxTokens: 1, errorText: "temperature 必须在 0 到 2 之间"},
		{name: "temperature is infinite", temperature: math.Inf(1), maxTokens: 1, errorText: "temperature 必须在 0 到 2 之间"},
		{name: "max tokens below minimum", temperature: 1, maxTokens: -1, errorText: "maxTokens 必须在 0 到 32768 之间"},
		{name: "max tokens above maximum", temperature: 1, maxTokens: 32769, errorText: "maxTokens 必须在 0 到 32768 之间"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateAIChatParams(&aiReq.AIChatParams{
				Temperature: &test.temperature,
				MaxTokens:   test.maxTokens,
			})
			assert.EqualError(t, err, test.errorText)
		})
	}
}

func TestValidateAIChatParamsRejectsNil(t *testing.T) {
	assert.EqualError(t, validateAIChatParams(nil), "聊天请求参数不能为空")
}
