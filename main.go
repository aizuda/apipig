package main

import (
	"apipig/core"
	"apipig/global"
	"apipig/initialize"
	"fmt"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// @title iota api
// @version dev
// @description artificial intelligence message push service
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name accessToken
// @BasePath /
func main() {
	configPath := core.ResolveConfigPath()
	if initialize.SetupRequired(configPath) {
		if err := initialize.RunSetup(configPath); err != nil {
			panic(err)
		}
	}

	core.Viper(configPath) // 初始化 Viper
	core.Zap()             // 初始化 Zap 日志库

	var port = global.CONFIG.System.Port
	fmt.Printf("IOTA - 物联网平台，专业、好用！ http://localhost:%d\n", port)
	fmt.Printf("SwaggerApi http://localhost:%d/swagger/index.html\n", port)

	global.DB = initialize.Gorm() // gorm连接数据库

	core.RunServer()
}
