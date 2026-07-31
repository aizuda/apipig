package config

import "strings"

type Plugin struct {

	// 协议网关插件
	ProtocolGateway ProtocolGateway `mapstructure:"protocol-gateway" json:"protocolGateway" yaml:"protocol-gateway"`

	// Emqx 插件
	Emqx Emqx `mapstructure:"emqx" json:"emqx" yaml:"emqx"`

	// TCP 插件
	Tcp Tcp `mapstructure:"tcp" json:"tcp" yaml:"tcp"`

	// Coap 插件
	Coap Coap `mapstructure:"coap" json:"coap" yaml:"coap"`
}

type ProtocolGateway struct {
	ListenNetwork string `mapstructure:"listen-network" json:"listenNetwork" yaml:"listen-network"` // 监听网络
	ListenAddress string `mapstructure:"listen-address" json:"listenAddress" yaml:"listen-address"` // 监听地址
	JwtKey        string `mapstructure:"jwt-key" json:"jwtKey" yaml:"jwt-key"`                      // jwt签名
	JwtExpires    int64  `mapstructure:"jwt-expires" json:"jwtExpires" yaml:"jwt-expires"`          // jwt过期时间（分钟）

}

type Emqx struct {
	Enable       bool   `mapstructure:"enable" json:"enable" yaml:"enable"`                     // 启用 true 是 false 否
	GrpcUrl      string `mapstructure:"grpc-url" json:"grpcUrl" yaml:"grpc-url"`                // ExHook gRPC 服务器地址 http / https 开头
	Server       string `mapstructure:"server" json:"server" yaml:"server"`                     // 服务地址 tcp://127.0.0.1:1883
	ClientId     string `mapstructure:"client-id" json:"clientId" yaml:"plugin-cid"`            // 客户端ID
	Username     string `mapstructure:"username" json:"username" yaml:"username"`               // 用户名
	Password     string `mapstructure:"password" json:"password" yaml:"password"`               // 密码
	PluginPrefix string `mapstructure:"plugin-prefix" json:"pluginPrefix" yaml:"plugin-prefix"` // 插件客户端ID前缀
	PluginToken  string `mapstructure:"plugin-token" json:"pluginToken" yaml:"plugin-token"`    // 插件客户端访问票据

}

func (m *Emqx) CheckPass(clientId, username, password string) bool {
	// 平台端验证
	return (clientId == m.ClientId && username == m.Username && password == m.Password) ||
		// 插件客户端验证，插件客户端ID = 插件客户端ID前缀 + 用户名，密码为插件客户端访问票据
		(strings.Contains(clientId, m.PluginPrefix) && clientId == (m.PluginPrefix+username) && password == m.PluginToken)
}

func (m *Emqx) IsPass(clientId, username string) bool {
	return (clientId == m.ClientId && username == m.Username) || strings.Contains(clientId, m.PluginPrefix)
}

type Tcp struct {
	Enable    bool   `mapstructure:"enable" json:"enable" yaml:"enable"`          // 启用 true 是 false 否
	Address   string `mapstructure:"address" json:"address" yaml:"address"`       // 服务地址
	Multicore bool   `mapstructure:"multicore" json:"multicore" yaml:"multicore"` // 多核处理器模式

}

type Coap struct {
	Enable  bool   `mapstructure:"enable" json:"enable" yaml:"enable"`    // 启用 true 是 false 否
	Address string `mapstructure:"address" json:"address" yaml:"address"` // 服务地址

}
