package core

import (
	"apipig/global"
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func Viper(path ...string) *viper.Viper {
	var config string
	if len(path) == 0 {
		flag.StringVar(&config, "c", "", "choose config file.")
		flag.Parse()
		if config == "" {
			// 优先级: 命令行 > 环境变量 > 默认值
			if configEnv := os.Getenv(global.ConfigEnv); configEnv == "" {
				config = global.ConfigFile
				fmt.Printf("您正在使用config的默认值,config的路径为%v\n", global.ConfigFile)
			} else {
				config = configEnv
				fmt.Printf("您正在使用CONFIG环境变量,config的路径为%v\n", config)
			}
		} else {
			fmt.Printf("您正在使用命令行的-c参数传递的值,config的路径为%v\n", config)
		}
	} else {
		config = path[0]
		fmt.Printf("您正在使用func Viper()传递的值,config的路径为%v\n", config)
	}

	v := viper.New()
	v.SetEnvPrefix("APIPIG")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	v.SetConfigFile(config)
	v.SetConfigType("yaml")

	// 设置默认值
	v.SetDefault("system.node", 0)
	v.SetDefault("system.aes-key", "18ffc810652aa83a")
	v.SetDefault("system.session-cron", "*/5 * * * *")
	v.SetDefault("system.print-route", false)
	v.SetDefault("system.enable-swagger", true)
	v.SetDefault("system.allow-origins", "*")
	v.SetDefault("device.check-offline-cron", "*/3 * * * *")
	v.SetDefault("device.check-offline-time", 10)
	v.SetDefault("device.data-relay-cron", "*/5 * * * *")
	v.SetDefault("device.data-relay-count", 10)
	v.SetDefault("database.max-idle-conns", 10)
	v.SetDefault("database.max-open-conns", 100)
	v.SetDefault("database.log-mode", "error")
	v.SetDefault("ai.log-queue-size", 2048)
	v.SetDefault("ai.log-batch-size", 100)
	v.SetDefault("ai.log-flush-interval-ms", 1000)
	v.SetDefault("ai.encryption-key", "")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件没有找到
			panic(fmt.Errorf("the config file does not exist: %s \n", err))
		} else {
			// 配置文件找到了,但是在这个过程有又出现别的什么error
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		if err := v.Unmarshal(&global.CONFIG); err != nil {
			fmt.Println(err)
		}
	})
	if err := v.Unmarshal(&global.CONFIG); err != nil {
		fmt.Println(err)
	}
	return v
}

func ResolveConfigPath() string {
	config := ""
	flag.StringVar(&config, "c", "", "choose config file.")
	flag.Parse()
	if config != "" {
		return config
	}
	if configEnv := os.Getenv(global.ConfigEnv); configEnv != "" {
		return configEnv
	}
	return global.ConfigFile
}
