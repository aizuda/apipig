package service

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"apipig/app/ai/model"
	"apipig/toolkit/snowflake"
)

type routeTarget struct {
	Provider model.Provider
	Channel  model.Channel
	Account  model.ChannelAccount
	Pricing  map[string]ModelPricingRule
	Proxy    *model.Proxy
}

func (s *GatewayService) pickRoute(modelName string) (routeTarget, error) {
	targets, err := s.loadRouteTargets()
	if err != nil {
		return routeTarget{}, err
	}
	var candidates []routeTarget
	selectedPriority := 0
	prioritySelected := false
	for _, target := range targets {
		providerModel, supported := resolveAccountModel(target.Account.Models, modelName)
		if s.breakerOpen(target.Channel.ID) || !supported || !containsModel(target.Provider.Models, providerModel) {
			continue
		}
		if !prioritySelected {
			selectedPriority = target.Channel.Priority
			prioritySelected = true
		}
		if target.Channel.Priority != selectedPriority {
			continue
		}
		candidates = append(candidates, target)
	}
	if len(candidates) == 0 {
		return routeTarget{}, errors.New("没有可用的模型渠道")
	}
	return weightedPick(candidates), nil
}

func (s *GatewayService) loadRouteTargets() ([]routeTarget, error) {
	const cacheTTL = 5 * time.Second
	s.routeMu.RLock()
	if !s.routeLoadedAt.IsZero() && time.Since(s.routeLoadedAt) < cacheTTL {
		targets := append([]routeTarget(nil), s.routeTargets...)
		s.routeMu.RUnlock()
		return targets, nil
	}
	s.routeMu.RUnlock()
	s.routeRefreshMu.Lock()
	defer s.routeRefreshMu.Unlock()
	s.routeMu.RLock()
	if !s.routeLoadedAt.IsZero() && time.Since(s.routeLoadedAt) < cacheTTL {
		targets := append([]routeTarget(nil), s.routeTargets...)
		s.routeMu.RUnlock()
		return targets, nil
	}
	s.routeMu.RUnlock()

	config, err := s.gatewayRepository().LoadRouteConfig()
	if err != nil {
		return nil, err
	}
	providerMap := make(map[snowflake.ID]model.Provider, len(config.Providers))
	for _, providerModel := range config.Providers {
		providerMap[providerModel.ID] = providerModel
	}

	proxyMap := make(map[snowflake.ID]model.Proxy, len(config.Proxies))
	for _, proxy := range config.Proxies {
		proxyMap[proxy.ID] = proxy
	}

	accountsByChannel := make(map[snowflake.ID][]model.ChannelAccount)
	for _, account := range config.Accounts {
		accountsByChannel[account.ChannelID] = append(accountsByChannel[account.ChannelID], account)
	}

	targets := make([]routeTarget, 0, len(config.Accounts))
	for _, channel := range config.Channels {
		providerModel, ok := providerMap[channel.ProviderID]
		if !ok {
			continue
		}
		for _, account := range accountsByChannel[channel.ID] {
			plaintextKey, err := s.credentialVault().Decrypt(account.APIKey)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(plaintextKey) == "" {
				continue
			}
			if s.credentialVault().Enabled() && !isEncryptedCredential(account.APIKey) {
				if encrypted, encryptErr := s.credentialVault().Encrypt(account.APIKey); encryptErr == nil {
					_ = s.gatewayRepository().UpdateChannelAccountAPIKey(account.ID, encrypted)
				}
			}
			account.APIKey = plaintextKey
			target := routeTarget{Provider: providerModel, Channel: channel, Account: account, Pricing: parseModelPricing(channel.ModelPricing)}
			if channel.ProxyID > 0 {
				if proxy, ok := proxyMap[channel.ProxyID]; ok {
					plaintextPassword, decryptErr := s.credentialVault().Decrypt(proxy.Password)
					if decryptErr != nil {
						return nil, decryptErr
					}
					if s.credentialVault().Enabled() && proxy.Password != "" && !isEncryptedCredential(proxy.Password) {
						if encrypted, encryptErr := s.credentialVault().Encrypt(proxy.Password); encryptErr == nil {
							_ = s.gatewayRepository().UpdateProxyPassword(proxy.ID, encrypted)
						}
					}
					proxy.Password = plaintextPassword
					target.Proxy = &proxy
				}
			}
			targets = append(targets, target)
		}
	}
	s.routeMu.Lock()
	s.routeLoadedAt = time.Now()
	s.routeTargets = append([]routeTarget(nil), targets...)
	s.routeMu.Unlock()
	return targets, nil
}

func (s *GatewayService) invalidateRouteTargets() {
	s.routeMu.Lock()
	s.routeLoadedAt = time.Time{}
	s.routeTargets = nil
	s.routeMu.Unlock()
}

func containsModel(models, modelName string) bool {
	models = strings.TrimSpace(models)
	modelName = strings.TrimSpace(modelName)
	if models == "" || modelName == "" {
		return true
	}
	if strings.HasPrefix(models, "{") {
		_, supported := resolveAccountModel(models, modelName)
		return supported
	}
	for _, item := range strings.Split(models, ",") {
		item = strings.TrimSpace(item)
		if item == "*" || item == modelName {
			return true
		}
	}
	return false
}

func weightedPick(candidates []routeTarget) routeTarget {
	if len(candidates) == 1 {
		return candidates[0]
	}
	total := 0
	for _, item := range candidates {
		if item.Channel.Weight > 0 {
			total += item.Channel.Weight
		}
	}
	if total <= 0 {
		return candidates[0]
	}
	n := rand.Intn(total)
	for _, item := range candidates {
		weight := item.Channel.Weight
		if weight <= 0 {
			continue
		}
		if n < weight {
			return item
		}
		n -= weight
	}
	return candidates[0]
}
