package service

import (
	"testing"

	"apipig/app/ai/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndValidateAccessTokenIPRule(t *testing.T) {
	token := model.AccessToken{
		IpRule: `{"enabled":true,"whitelist":["192.168.1.10, 10.0.0.0/8","192.168.1.10"],"blacklist":["203.0.113.8"]}`,
	}

	require.NoError(t, normalizeAccessTokenIPRule(&token))
	require.NoError(t, validateAccessTokenIPRule(token.IpRule))
	assert.JSONEq(t, `{
		"enabled": true,
		"whitelist": ["192.168.1.10", "10.0.0.0/8"],
		"blacklist": ["203.0.113.8"]
	}`, token.IpRule)
}

func TestNormalizeEmptyAccessTokenIPRule(t *testing.T) {
	token := model.AccessToken{}
	require.NoError(t, normalizeAccessTokenIPRule(&token))
	assert.JSONEq(t, `{"enabled":false,"whitelist":[],"blacklist":[]}`, token.IpRule)
}

func TestValidateAccessTokenIPRuleRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid json", value: `{"enabled":`},
		{name: "empty enabled rule", value: `{"enabled":true,"whitelist":[],"blacklist":[]}`},
		{name: "invalid whitelist", value: `{"enabled":true,"whitelist":["300.1.1.1"],"blacklist":[]}`},
		{name: "invalid blacklist", value: `{"enabled":true,"whitelist":[],"blacklist":["10.0.0.0/99"]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, validateAccessTokenIPRule(test.value))
		})
	}
}

func TestValidateAccessTokenIPRuleIgnoresDisabledEntries(t *testing.T) {
	require.NoError(t, validateAccessTokenIPRule(
		`{"enabled":false,"whitelist":["not-an-ip"],"blacklist":[]}`,
	))
}

func TestAccessTokenAllowsIP(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		clientIP string
		allowed  bool
	}{
		{name: "empty rule allows", rule: "", clientIP: "203.0.113.8", allowed: true},
		{name: "disabled rule allows", rule: `{"enabled":false,"whitelist":["10.0.0.0/8"],"blacklist":["203.0.113.8"]}`, clientIP: "203.0.113.8", allowed: true},
		{name: "whitelist allows match", rule: `{"enabled":true,"whitelist":["10.0.0.0/8"],"blacklist":[]}`, clientIP: "10.12.3.4", allowed: true},
		{name: "whitelist rejects unmatched", rule: `{"enabled":true,"whitelist":["10.0.0.0/8"],"blacklist":[]}`, clientIP: "192.168.1.1", allowed: false},
		{name: "blacklist rejects match", rule: `{"enabled":true,"whitelist":[],"blacklist":["203.0.113.8"]}`, clientIP: "203.0.113.8", allowed: false},
		{name: "blacklist allows unmatched", rule: `{"enabled":true,"whitelist":[],"blacklist":["203.0.113.8"]}`, clientIP: "203.0.113.9", allowed: true},
		{name: "blacklist wins over whitelist", rule: `{"enabled":true,"whitelist":["203.0.113.0/24"],"blacklist":["203.0.113.8"]}`, clientIP: "203.0.113.8", allowed: false},
		{name: "ipv6 whitelist match", rule: `{"enabled":true,"whitelist":["2001:db8::/32"],"blacklist":[]}`, clientIP: "2001:db8::1", allowed: true},
		{name: "invalid json fails closed", rule: `{"enabled":`, clientIP: "127.0.0.1", allowed: false},
		{name: "enabled rule rejects invalid client IP", rule: `{"enabled":true,"whitelist":["127.0.0.1"],"blacklist":[]}`, clientIP: "", allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := model.AccessToken{IpRule: test.rule}
			assert.Equal(t, test.allowed, accessTokenAllowsIP(token, test.clientIP))
		})
	}
}
