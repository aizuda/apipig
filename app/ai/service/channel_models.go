package service

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

func parseAccountModelMappings(value string) (map[string]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]string{}, nil
	}
	mappings := make(map[string]string)
	if strings.HasPrefix(value, "{") {
		if err := json.Unmarshal([]byte(value), &mappings); err != nil {
			return nil, errors.New("账户模型映射 JSON 格式错误")
		}
	} else {
		for _, modelName := range splitModels(value) {
			mappings[modelName] = modelName
		}
	}
	normalized := make(map[string]string, len(mappings))
	for gatewayModel, providerModel := range mappings {
		gatewayModel = strings.TrimSpace(gatewayModel)
		providerModel = strings.TrimSpace(providerModel)
		if gatewayModel == "" || providerModel == "" {
			return nil, errors.New("账户模型映射名称不能为空")
		}
		normalized[gatewayModel] = providerModel
	}
	return normalized, nil
}

func normalizeAccountModels(value string) (string, error) {
	mappings, err := parseAccountModelMappings(value)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(mappings)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func resolveAccountModel(value, gatewayModel string) (string, bool) {
	mappings, err := parseAccountModelMappings(value)
	if err != nil || len(mappings) == 0 {
		return gatewayModel, err == nil
	}
	if providerModel, ok := mappings[gatewayModel]; ok {
		return providerModel, true
	}
	if providerModel, ok := mappings["*"]; ok {
		if providerModel == "*" {
			return gatewayModel, true
		}
		return providerModel, true
	}
	return "", false
}

func accountGatewayModels(value string) ([]string, error) {
	mappings, err := parseAccountModelMappings(value)
	if err != nil || len(mappings) == 0 {
		return nil, err
	}
	models := make([]string, 0, len(mappings))
	for gatewayModel := range mappings {
		models = append(models, gatewayModel)
	}
	sort.Strings(models)
	return models, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
