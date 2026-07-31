package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"unicode"

	"apipig/app/ai/model"
)

type accessTokenIPRule struct {
	Enabled   bool     `json:"enabled"`
	Whitelist []string `json:"whitelist"`
	Blacklist []string `json:"blacklist"`
}

func normalizeAccessTokenIPRule(token *model.AccessToken) error {
	rule, err := parseAccessTokenIPRuleJSON(token.IpRule)
	if err != nil {
		return err
	}
	rule.Whitelist = normalizeAccessTokenIPRules(rule.Whitelist)
	rule.Blacklist = normalizeAccessTokenIPRules(rule.Blacklist)
	normalized, err := json.Marshal(rule)
	if err != nil {
		return errors.New("API 密钥 IP 规则序列化失败")
	}
	token.IpRule = string(normalized)
	return nil
}

func validateAccessTokenIPRule(value string) error {
	rule, err := parseAccessTokenIPRuleJSON(value)
	if err != nil {
		return err
	}
	if !rule.Enabled {
		return nil
	}
	if rule.Enabled && len(rule.Whitelist) == 0 && len(rule.Blacklist) == 0 {
		return errors.New("API 密钥启用 IP 限制时白名单和黑名单不能同时为空")
	}
	for _, item := range append(append([]string{}, rule.Whitelist...), rule.Blacklist...) {
		if _, _, parseErr := parseAccessTokenIPEntry(item); parseErr != nil {
			return fmt.Errorf("API 密钥 IP 或 CIDR 格式无效：%s", item)
		}
	}
	return nil
}

func accessTokenAllowsIP(token model.AccessToken, clientIP string) bool {
	if err := validateAccessTokenIPRule(token.IpRule); err != nil {
		return false
	}
	rule, err := parseAccessTokenIPRuleJSON(token.IpRule)
	if err != nil || !rule.Enabled {
		return err == nil
	}
	clientAddr, err := netip.ParseAddr(strings.TrimSpace(clientIP))
	if err != nil {
		return false
	}
	clientAddr = clientAddr.Unmap()

	if accessTokenIPListContains(rule.Blacklist, clientAddr) {
		return false
	}
	if len(rule.Whitelist) == 0 {
		return true
	}
	return accessTokenIPListContains(rule.Whitelist, clientAddr)
}

func parseAccessTokenIPRuleJSON(value string) (accessTokenIPRule, error) {
	rule := accessTokenIPRule{
		Whitelist: make([]string, 0),
		Blacklist: make([]string, 0),
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return rule, nil
	}
	if err := json.Unmarshal([]byte(value), &rule); err != nil {
		return accessTokenIPRule{}, errors.New("API 密钥 IP 规则必须是有效的 JSON")
	}
	rule.Whitelist = normalizeAccessTokenIPRules(rule.Whitelist)
	rule.Blacklist = normalizeAccessTokenIPRules(rule.Blacklist)
	return rule, nil
}

func normalizeAccessTokenIPRules(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, item := range splitAccessTokenIPEntries(value) {
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func splitAccessTokenIPEntries(value string) []string {
	parts := strings.FieldsFunc(value, func(char rune) bool {
		return char == ',' || char == ';' || unicode.IsSpace(char)
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func accessTokenIPListContains(values []string, clientAddr netip.Addr) bool {
	for _, value := range values {
		address, prefix, err := parseAccessTokenIPEntry(value)
		if err != nil {
			return false
		}
		if (prefix.IsValid() && prefix.Contains(clientAddr)) || (address.IsValid() && address == clientAddr) {
			return true
		}
	}
	return false
}

func parseAccessTokenIPEntry(value string) (netip.Addr, netip.Prefix, error) {
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Addr{}, netip.Prefix{}, err
		}
		return netip.Addr{}, prefix.Masked(), nil
	}
	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, netip.Prefix{}, err
	}
	return address.Unmap(), netip.Prefix{}, nil
}
