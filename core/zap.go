package core

import (
	"apipig/global"
	"encoding/json"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

func Zap() {
	// 读取配置文件
	configFile, err := os.ReadFile("logger.json")
	if err != nil {
		panic(err)
	}

	var cfg zap.Config
	if err := json.Unmarshal(configFile, &cfg); err != nil {
		panic(err)
	}
	// 设置日志和错误日志的 lumberjack 配置
	if len(cfg.OutputPaths) < 2 {
		panic("Need at least two paths for logs and error logs respectively")
	}
	// 设置 lumberjack 作为日志文件的滚动处理
	logWriter := addSyncLumberjack(cfg.OutputPaths[1])

	// 创建 logger
	encoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)
	core := zapcore.NewCore(encoder, logWriter, cfg.Level)

	logger := zap.New(core)
	defer logger.Sync()

	global.LOG = logger
}

func addSyncLumberjack(filename string) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   filename,
		MaxSize:    30,   // 文件大小限制，单位是 MB
		MaxBackups: 5,    // 最多保留 5 个备份
		MaxAge:     30,   // 文件保留最大天数
		Compress:   true, // 是否压缩/归档旧文件
	})
}
