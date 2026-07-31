package router

type AiRouterGroup struct {
	GatewayRouter
	ProviderRouter
	ChannelRouter
	ChannelAccountRouter
	AccessTokenRouter
	AccessTokenTagRouter
	ProxyRouter
	CallLogRouter
}

var AiRouter = new(AiRouterGroup)
