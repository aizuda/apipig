package response

import "apipig/app/ai/model"

// ChannelPageRecord 表示渠道分页列表所需的渠道及关联展示信息。
type ChannelPageRecord struct {
	model.Channel
	ProviderName    string   `json:"providerName"`
	ProxyName       string   `json:"proxyName"`
	AvailableModels []string `json:"availableModels"`
	ProviderModels  []string `json:"providerModels"`
}

// ChannelAccountPageRecord 表示账户分页列表所需的账户及关联展示信息。
type ChannelAccountPageRecord struct {
	model.ChannelAccount
	ChannelName  string `json:"channelName"`
	ProviderName string `json:"providerName"`
}
