package initialize

import (
	"apipig/config"
	"apipig/toolkit"
	buildversion "apipig/version"
	"apipig/web"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v2"
)

const setupPort = 9527

const defaultSetupLoggerConfig = `{
  "level": "error",
  "encoding": "json",
  "outputPaths": ["stdout", "./logs/apipig.log"],
  "encoderConfig": {
    "messageKey": "message",
    "levelKey": "level",
    "timeKey": "time",
    "levelEncoder": "lowercase",
    "timeEncoder": "iso8601"
  }
}
`

type setupRequest struct {
	Version       string `json:"version"`
	Port          int    `json:"port"`
	DBType        string `json:"dbType"`
	SQLitePath    string `json:"sqlitePath"`
	Host          string `json:"host"`
	DatabasePort  string `json:"databasePort"`
	DBName        string `json:"dbName"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	DBConfig      string `json:"dbConfig"`
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`
	MaxIdleConns  int    `json:"maxIdleConns"`
	MaxOpenConns  int    `json:"maxOpenConns"`
	LogMode       string `json:"logMode"`
	EnableSwagger bool   `json:"enableSwagger"`
	PrintRoute    bool   `json:"printRoute"`
	AllowOrigins  string `json:"allowOrigins"`
}

type setupConfig struct {
	Version     string             `yaml:"version"`
	System      config.System      `yaml:"system"`
	JWT         config.JWT         `yaml:"jwt"`
	AI          config.AI          `yaml:"ai"`
	CodeReview  config.CodeReview  `yaml:"code-review"`
	RemoteAgent config.RemoteAgent `yaml:"remote-agent"`
	Database    config.Database    `yaml:"database"`
}

type setupResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

func SetupRequired(configPath string) bool {
	info, err := os.Stat(configPath)
	return errors.Is(err, os.ErrNotExist) || err == nil && info.Size() == 0
}

func RunSetup(configPath string) error {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	app := newSetupApp(absConfigPath)
	address := fmt.Sprintf(":%d", setupPort)
	setupURL := fmt.Sprintf("http://localhost:%d/", setupPort)
	fmt.Printf("首次运行，请在浏览器中完成初始化：%s\n", setupURL)
	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = openBrowser(setupURL)
	}()
	return app.Listen(address)
}

func newSetupApp(configPath string) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	var submitted atomic.Bool

	app.Get("/", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.SendString(web.InitHTML)
	})
	app.Get("/api/init/meta", func(c *fiber.Ctx) error {
		return c.JSON(setupResponse{Code: "1", Msg: "ok", Data: fiber.Map{
			"configPath": configPath,
			"defaults":   defaultSetupRequest(),
		}})
	})
	app.Post("/api/init/config", func(c *fiber.Ctx) error {
		if !submitted.CompareAndSwap(false, true) {
			return c.Status(fiber.StatusConflict).JSON(setupResponse{Code: "0", Msg: "初始化正在执行，请勿重复提交"})
		}

		var request setupRequest
		if err := c.BodyParser(&request); err != nil {
			submitted.Store(false)
			return c.Status(fiber.StatusBadRequest).JSON(setupResponse{Code: "0", Msg: "初始化参数格式错误"})
		}
		request.applyDefaults()
		if err := request.validate(); err != nil {
			submitted.Store(false)
			return c.Status(fiber.StatusBadRequest).JSON(setupResponse{Code: "0", Msg: err.Error()})
		}

		generated, err := request.buildConfig()
		if err == nil {
			err = initializeSetupDatabase(generated.Database, request.AdminUsername, request.AdminPassword)
		}
		if err == nil {
			err = writeSetupLoggerConfig("logger.json")
		}
		if err == nil {
			err = writeSetupConfig(configPath, generated)
		}
		if err != nil {
			submitted.Store(false)
			return c.Status(fiber.StatusBadRequest).JSON(setupResponse{Code: "0", Msg: "初始化失败：" + err.Error()})
		}

		redirectPath := setupRedirectURL(c.Hostname(), request.Port)
		responseErr := c.JSON(setupResponse{Code: "1", Msg: "初始化成功", Data: fiber.Map{
			"configPath":   configPath,
			"redirectPath": redirectPath,
			"autoRestart":  true,
		}})
		go func() {
			time.Sleep(300 * time.Millisecond)
			_ = app.Shutdown()
		}()
		return responseErr
	})
	app.Use(func(c *fiber.Ctx) error {
		return c.Redirect("/", fiber.StatusFound)
	})
	return app
}

func defaultSetupRequest() setupRequest {
	return setupRequest{
		Version:       buildversion.Version,
		Port:          setupPort,
		DBType:        "sqlite",
		SQLitePath:    "apipig.db?_busy_timeout=30000",
		Host:          "127.0.0.1",
		DatabasePort:  "5432",
		DBName:        "apipig",
		Username:      "postgres",
		AdminUsername: "admin",
		MaxIdleConns:  10,
		MaxOpenConns:  100,
		LogMode:       "error",
		AllowOrigins:  "*",
	}
}

func (request *setupRequest) applyDefaults() {
	defaults := defaultSetupRequest()
	request.Version = valueOrDefault(request.Version, defaults.Version)
	request.DBType = strings.ToLower(valueOrDefault(request.DBType, defaults.DBType))
	request.AdminUsername = valueOrDefault(request.AdminUsername, defaults.AdminUsername)
	request.AllowOrigins = valueOrDefault(request.AllowOrigins, defaults.AllowOrigins)
	request.LogMode = strings.ToLower(valueOrDefault(request.LogMode, defaults.LogMode))
	if request.Port == 0 {
		request.Port = defaults.Port
	}
	if request.MaxIdleConns == 0 {
		request.MaxIdleConns = defaults.MaxIdleConns
	}
	if request.MaxOpenConns == 0 {
		request.MaxOpenConns = defaults.MaxOpenConns
	}
	if request.DBType == "sqlite" {
		request.SQLitePath = valueOrDefault(request.SQLitePath, defaults.SQLitePath)
		return
	}
	request.Host = valueOrDefault(request.Host, defaults.Host)
	request.DBName = valueOrDefault(request.DBName, defaults.DBName)
	if request.DBType == "mysql" {
		request.DatabasePort = valueOrDefault(request.DatabasePort, "3306")
		request.Username = valueOrDefault(request.Username, "root")
		request.DBConfig = valueOrDefault(request.DBConfig, "charset=utf8mb4&parseTime=True&loc=Local")
		return
	}
	request.DatabasePort = valueOrDefault(request.DatabasePort, defaults.DatabasePort)
	request.Username = valueOrDefault(request.Username, defaults.Username)
	request.DBConfig = valueOrDefault(request.DBConfig, "sslmode=disable TimeZone=Asia/Shanghai")
}

func (request setupRequest) validate() error {
	if request.Port < 1 || request.Port > 65535 {
		return errors.New("服务端口必须在 1 到 65535 之间")
	}
	if request.DBType != "sqlite" && request.DBType != "mysql" && request.DBType != "postgresql" {
		return errors.New("仅支持 sqlite、mysql 或 postgresql 数据库")
	}
	if request.DBType == "sqlite" && request.SQLitePath == "" {
		return errors.New("SQLite 路径不能为空")
	}
	if request.DBType != "sqlite" && (request.Host == "" || request.DatabasePort == "" || request.DBName == "" || request.Username == "") {
		return errors.New("数据库地址、端口、名称和用户名不能为空")
	}
	if request.AdminUsername == "" || len(request.AdminUsername) > 30 || strings.ContainsAny(request.AdminUsername, " \t\r\n") {
		return errors.New("管理员账号不能为空、不能包含空白且长度不能超过 30 个字符")
	}
	if len(request.AdminPassword) < 6 {
		return errors.New("管理员密码长度不能少于 6 个字符")
	}
	if request.MaxIdleConns < 1 || request.MaxOpenConns < request.MaxIdleConns {
		return errors.New("数据库连接池参数无效")
	}
	switch request.LogMode {
	case "silent", "error", "warn", "info":
		return nil
	default:
		return errors.New("数据库日志级别无效")
	}
}

func (request setupRequest) buildConfig() (setupConfig, error) {
	privateKey, publicKey, err := toolkit.GenerateRSAKeyPair(2048)
	if err != nil {
		return setupConfig{}, err
	}
	signingKey, err := secureRandomString(48)
	if err != nil {
		return setupConfig{}, err
	}
	aesKey, err := secureRandomString(32)
	if err != nil {
		return setupConfig{}, err
	}
	aiKey, err := secureRandomString(48)
	if err != nil {
		return setupConfig{}, err
	}

	database := config.Database{
		DbType:       request.DBType,
		SQLitePath:   request.SQLitePath,
		Host:         request.Host,
		Port:         request.DatabasePort,
		Dbname:       request.DBName,
		Username:     request.Username,
		Password:     request.Password,
		Config:       request.DBConfig,
		MaxIdleConns: request.MaxIdleConns,
		MaxOpenConns: request.MaxOpenConns,
		LogMode:      request.LogMode,
	}
	if request.DBType == "sqlite" {
		database.Host = ""
		database.Port = ""
		database.Dbname = ""
		database.Username = ""
		database.Password = ""
		database.Config = ""
	}

	return setupConfig{
		Version: request.Version,
		System: config.System{
			Port:          request.Port,
			Version:       request.Version,
			DbType:        request.DBType,
			ContextPath:   "",
			AesKey:        aesKey,
			SessionCron:   "*/5 * * * *",
			PrintRoute:    request.PrintRoute,
			EnableSwagger: request.EnableSwagger,
			AllowOrigins:  request.AllowOrigins,
		},
		JWT: config.JWT{
			SigningKey:  signingKey,
			ExpiresTime: 30,
			PrivateKey:  base64.StdEncoding.EncodeToString(privateKey),
			PublicKey:   publicKey,
		},
		AI: config.AI{
			EncryptionKey:      aiKey,
			LogQueueSize:       2048,
			LogBatchSize:       100,
			LogFlushIntervalMs: 1000,
		},
		CodeReview: config.CodeReview{
			WorkerCount:     2,
			QueueSize:       128,
			GitTimeoutSec:   120,
			AITimeoutSec:    180,
			MaxDiffBytes:    524288,
			MaxChangedFiles: 200,
		},
		RemoteAgent: config.RemoteAgent{
			HeartbeatTimeoutSeconds:   30,
			CommandPollTimeoutSeconds: 25,
			DispatchLeaseSeconds:      30,
		},
		Database: database,
	}, nil
}

func initializeSetupDatabase(database config.Database, username, password string) error {
	db, err := OpenDatabase(database)
	if err != nil {
		return fmt.Errorf("连接数据库: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接: %w", err)
	}
	defer sqlDB.Close()
	if err = sqlDB.Ping(); err != nil {
		return fmt.Errorf("验证数据库连接: %w", err)
	}
	configureConnectionPool(database, db)
	if err = initData(db, &AdminAccount{Username: username, Password: password}); err != nil {
		return fmt.Errorf("初始化数据库: %w", err)
	}
	return nil
}

func writeSetupConfig(configPath string, generated setupConfig) error {
	content, err := yaml.Marshal(generated)
	if err != nil {
		return fmt.Errorf("生成配置: %w", err)
	}
	directory := filepath.Dir(configPath)
	if err = os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("创建配置目录: %w", err)
	}
	if info, statErr := os.Stat(configPath); statErr == nil && info.Size() > 0 {
		return errors.New("配置文件已存在，拒绝覆盖")
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("检查配置文件: %w", statErr)
	}

	temporary, err := os.CreateTemp(directory, ".apipig-config-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时配置: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err = temporary.Chmod(0600); err == nil {
		_, err = temporary.Write(content)
	}
	if err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err != nil {
		return fmt.Errorf("写入配置: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭配置文件: %w", closeErr)
	}
	if info, statErr := os.Stat(configPath); statErr == nil && info.Size() == 0 {
		if err = os.Remove(configPath); err != nil {
			return fmt.Errorf("替换空配置文件: %w", err)
		}
	}
	if err = os.Rename(temporaryPath, configPath); err != nil {
		return fmt.Errorf("保存配置: %w", err)
	}
	return nil
}

func writeSetupLoggerConfig(loggerPath string) error {
	directory := filepath.Dir(loggerPath)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("创建日志配置目录: %w", err)
	}
	if info, statErr := os.Stat(loggerPath); statErr == nil && info.Size() > 0 {
		return nil
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("检查日志配置文件: %w", statErr)
	}

	temporary, err := os.CreateTemp(directory, ".apipig-logger-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时日志配置: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err = temporary.Chmod(0644); err == nil {
		_, err = temporary.WriteString(defaultSetupLoggerConfig)
	}
	if err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err != nil {
		return fmt.Errorf("写入日志配置: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭日志配置文件: %w", closeErr)
	}
	if info, statErr := os.Stat(loggerPath); statErr == nil {
		if info.Size() > 0 {
			return nil
		}
		if err = os.Remove(loggerPath); err != nil {
			return fmt.Errorf("替换空日志配置文件: %w", err)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("检查日志配置文件: %w", statErr)
	}
	if err = os.Rename(temporaryPath, loggerPath); err != nil {
		return fmt.Errorf("保存日志配置: %w", err)
	}
	return nil
}

func secureRandomString(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func setupRedirectURL(host string, port int) string {
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, fmt.Sprintf("%d", port)), Path: "/"}).String()
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func openBrowser(target string) error {
	if os.Getenv("APIPIG_NO_BROWSER") == "1" {
		return nil
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		command = exec.Command("open", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}
