package api

import "apipig/app/ai/service"

type AiApiGroup struct {
	GatewayApi        *GatewayApi
	ProviderApi       *ProviderApi
	ChannelApi        *ChannelApi
	ChannelAccountApi *ChannelAccountApi
	AccessTokenApi    *AccessTokenApi
	AccessTokenTagApi *AccessTokenTagApi
	ProxyApi          *ProxyApi
	CallLogApi        *CallLogApi
}

func NewAiApiGroup(services *service.AiServiceGroup) *AiApiGroup {
	return &AiApiGroup{
		GatewayApi:        NewGatewayApi(services.GatewayService),
		ProviderApi:       NewProviderApi(services.ProviderService),
		ChannelApi:        NewChannelApi(services.ChannelService),
		ChannelAccountApi: NewChannelAccountApi(services.ChannelAccountService),
		AccessTokenApi:    NewAccessTokenApi(services.AccessTokenService),
		AccessTokenTagApi: NewAccessTokenTagApi(services.AccessTokenTagService),
		ProxyApi:          NewProxyApi(services.ProxyService),
		CallLogApi:        NewCallLogApi(services.CallLogService),
	}
}

var AiApi = NewAiApiGroup(service.AiService)
