package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"apipig/app/ai/model"
)

const (
	maxGatewayRequestBodyBytes            = 8 << 20
	maxAudioTranscriptionFileBytes        = 25 << 20
	maxAudioTranscriptionRequestBodyBytes = maxAudioTranscriptionFileBytes + (1 << 20)
	maxGatewayResponseBodyBytes           = 32 << 20
	maxGatewayMessages                    = 256
	maxGatewayTools                       = 128
	maxGatewayContentParts                = 256
	maxModelPricingBytes                  = 64 << 10
)

func validateProvider(m *model.Provider) error {
	if m.Name == "" || m.Code == "" {
		return errors.New("供应商名称和编码不能为空")
	}
	if err := validateTextFields(
		textField{m.Name, 80, "供应商名称"}, textField{m.Code, 50, "供应商编码"},
		textField{m.Icon, 50, "供应商图标标识"}, textField{m.Protocol, 30, "供应商协议"},
		textField{m.BaseURL, 255, "供应商 BaseURL"}, textField{m.Models, 1000, "供应商模型列表"},
		textField{m.Remark, 255, "供应商备注"},
	); err != nil {
		return err
	}
	parsed, err := url.Parse(m.BaseURL)
	if err != nil || parsed.Hostname() == "" || !slices.Contains([]string{"http", "https"}, strings.ToLower(parsed.Scheme)) {
		return errors.New("供应商 BaseURL 必须是有效的 HTTP 或 HTTPS 地址")
	}
	if parsed.User != nil {
		return errors.New("供应商 BaseURL 不允许包含用户名或密码")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("供应商 BaseURL 不允许包含查询参数或片段")
	}
	if m.TimeoutMs < 1000 || m.TimeoutMs > 10*60*1000 {
		return errors.New("供应商超时时间必须在 1000 到 600000 毫秒之间")
	}
	return nil
}

func validateChannel(m *model.Channel, store AIStore) error {
	if m.ProviderID == 0 || m.Name == "" {
		return errors.New("渠道供应商和名称不能为空")
	}
	if err := validateTextFields(
		textField{m.Name, 80, "渠道名称"}, textField{m.Remark, 255, "渠道备注"},
	); err != nil {
		return err
	}
	if err := validateByteFields(byteField{m.ModelPricing, maxModelPricingBytes, "渠道计价配置"}); err != nil {
		return err
	}
	if m.RPM < 0 || m.TPM < 0 {
		return errors.New("渠道 RPM 和 TPM 不能为负数")
	}
	if m.Weight < 1 || m.Weight > 10000 {
		return errors.New("渠道权重必须在 1 到 10000 之间")
	}
	if m.CostMultiplier > 1000 {
		return errors.New("渠道成本倍率不能超过 1000")
	}
	var providerModel model.Provider
	if err := resolveAIStore(store).GetByID(&providerModel, m.ProviderID); err != nil {
		return errors.New("渠道关联的供应商不存在")
	}
	if m.ProxyID > 0 {
		var proxy model.Proxy
		if err := resolveAIStore(store).GetByID(&proxy, m.ProxyID); err != nil {
			return errors.New("渠道关联的代理不存在")
		}
	}
	for pricingModel := range parseModelPricing(m.ModelPricing) {
		if pricingModel != "*" && !containsModel(providerModel.Models, pricingModel) {
			return fmt.Errorf("计价模型 %s 不在供应商支持模型中", pricingModel)
		}
	}
	return nil
}

func validateAccessToken(m *model.AccessToken, store AIStore) error {
	if m.Name == "" {
		return errors.New("访问 Token 名称不能为空")
	}
	if err := validateTextFields(
		textField{m.Name, 80, "访问 Token 名称"}, textField{m.Models, 1000, "访问 Token 模型列表"},
		textField{m.Remark, 255, "访问 Token 备注"},
	); err != nil {
		return err
	}
	if err := validateByteFields(
		byteField{m.IpRule, maxModelPricingBytes, "API 密钥 IP 规则"},
		byteField{m.RateLimitRule, 4096, "API 密钥消费限制规则"},
	); err != nil {
		return err
	}
	if m.RPM < 0 || m.TPM < 0 {
		return errors.New("访问 Token 限额不能为负数")
	}
	if err := validateUSD(m.QuotaAmount, "访问 Token 成本额度"); err != nil {
		return err
	}
	if err := validateAccessTokenIPRule(m.IpRule); err != nil {
		return err
	}
	if err := validateAccessTokenRateLimitRule(m.RateLimitRule); err != nil {
		return err
	}
	if m.ExpireAt < 0 {
		return errors.New("API 密钥过期时间不能为负数")
	}
	if m.ChannelID == 0 {
		return errors.New("API 密钥关联渠道不能为空")
	}
	var channel model.Channel
	if err := resolveAIStore(store).GetByID(&channel, m.ChannelID); err != nil {
		return errors.New("API 密钥关联的渠道不存在")
	}
	selectedModels := splitModels(m.Models)
	if len(selectedModels) == 0 {
		return errors.New("API 密钥关联渠道模型不能为空")
	}
	var accounts []model.ChannelAccount
	if err := resolveAIStore(store).Query(model.ChannelAccount{}).
		Where("channel_id = ? AND status = ?", channel.ID, gatewayStatusNormal).
		Find(&accounts).Error; err != nil {
		return err
	}
	availableModels := make([]string, 0)
	for _, account := range accounts {
		models, err := accountGatewayModels(account.Models)
		if err != nil {
			return err
		}
		availableModels = append(availableModels, models...)
	}
	availableModels = uniqueStrings(availableModels)
	availableModelSet := make(map[string]struct{}, len(availableModels))
	for _, modelName := range availableModels {
		availableModelSet[modelName] = struct{}{}
	}
	for _, modelName := range selectedModels {
		if _, wildcard := availableModelSet["*"]; wildcard {
			continue
		}
		if _, exists := availableModelSet[modelName]; !exists {
			return fmt.Errorf("API 密钥模型 %s 不在关联号池可用模型中", modelName)
		}
	}
	return nil
}

func validateProxy(m *model.Proxy) error {
	if m.Name == "" || m.Host == "" {
		return errors.New("代理名称和主机不能为空")
	}
	if err := validateTextFields(
		textField{m.Name, 80, "代理名称"}, textField{m.Host, 120, "代理主机"},
		textField{m.Username, 120, "代理用户名"},
		textField{m.Region, 80, "代理区域"}, textField{m.Remark, 255, "代理备注"},
	); err != nil {
		return err
	}
	if err := validateByteFields(byteField{m.Password, 1400, "代理密码"}); err != nil {
		return err
	}
	if !slices.Contains([]string{"http", "https", "socks5"}, m.Scheme) {
		return fmt.Errorf("不支持的代理协议: %s", m.Scheme)
	}
	if m.Username == "" && m.Password != "" {
		return errors.New("代理设置认证密码时必须同时填写用户名")
	}
	if err := validateProxyHost(m.Host); err != nil {
		return err
	}
	if m.Port < 1 || m.Port > 65535 {
		return errors.New("代理端口必须在 1 到 65535 之间")
	}
	return nil
}

type textField struct {
	value string
	max   int
	name  string
}

type byteField struct {
	value string
	max   int
	name  string
}

func validateTextFields(fields ...textField) error {
	for _, field := range fields {
		if utf8.RuneCountInString(field.value) > field.max {
			return fmt.Errorf("%s不能超过 %d 个字符", field.name, field.max)
		}
	}
	return nil
}

func validateByteFields(fields ...byteField) error {
	for _, field := range fields {
		if len([]byte(field.value)) > field.max {
			return fmt.Errorf("%s不能超过 %d 字节", field.name, field.max)
		}
	}
	return nil
}

func validateProxyHost(host string) error {
	host = strings.Trim(host, "[]")
	if net.ParseIP(host) != nil {
		return nil
	}
	if host == "" || len(host) > 253 {
		return errors.New("代理主机必须是有效的 IP 地址或域名")
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return errors.New("代理主机必须是有效的 IP 地址或域名")
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return errors.New("代理主机必须是有效的 IP 地址或域名")
			}
		}
	}
	return nil
}

func normalizeModels(value string) string {
	seen := make(map[string]struct{})
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	return strings.Join(items, ",")
}
