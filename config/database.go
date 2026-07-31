package config

import (
	"apipig/toolkit"
	"fmt"
)

type Database struct {
	DbType       string `mapstructure:"db-type" json:"dbType" yaml:"db-type"`                       // 数据库类型 sqlite mysql postgresql
	SQLitePath   string `mapstructure:"sqlite-path" json:"sqlitePath" yaml:"sqlite-path,omitempty"` // SQLite 路径或 DSN
	Host         string `mapstructure:"host" json:"host" yaml:"host"`                               // 服务器地址
	Port         string `mapstructure:"port" json:"port" yaml:"port"`                               // 数据库端口
	Dbname       string `mapstructure:"db-name" json:"dbname" yaml:"db-name"`                       // 数据库名
	Username     string `mapstructure:"username" json:"username" yaml:"username"`                   // 数据库用户名
	Password     string `mapstructure:"password" json:"password" yaml:"password"`                   // 数据库密码
	Config       string `mapstructure:"config" json:"config" yaml:"config"`                         // 高级配置
	MaxIdleConns int    `mapstructure:"max-idle-conns" json:"maxIdleConns" yaml:"max-idle-conns"`   // 空闲中的最大连接数
	MaxOpenConns int    `mapstructure:"max-open-conns" json:"maxOpenConns" yaml:"max-open-conns"`   // 打开到数据库的最大连接数
	LogMode      string `mapstructure:"log-mode" json:"logMode" yaml:"log-mode"`                    // 是否开启Gorm全局日志
}

type TimeSeriesDb struct {
	DbType           string `mapstructure:"db-type" json:"dbType" yaml:"db-type"`                                 // 数据库类型 queryDb
	Host             string `mapstructure:"host" json:"host" yaml:"host"`                                         // 服务器地址
	Port             string `mapstructure:"port" json:"port" yaml:"port"`                                         // 数据库端口
	Dbname           string `mapstructure:"db-name" json:"dbname" yaml:"db-name"`                                 // 数据库名
	Username         string `mapstructure:"username" json:"username" yaml:"username"`                             // 数据库用户名
	Password         string `mapstructure:"password" json:"password" yaml:"password"`                             // 数据库密码
	Config           string `mapstructure:"config" json:"config" yaml:"config"`                                   // 高级配置
	DudTableStrategy uint   `mapstructure:"dud-table-strategy" json:"dudTableStrategy" yaml:"dud-table-strategy"` // 设备上报数据表后缀策略 1，年 2，月 3，日 其它按照日策略处理
	dts              uint   // 缓存设备上报数据判断策略
	DdmTableStrategy uint   `mapstructure:"ddm-table-strategy" json:"ddmTableStrategy" yaml:"ddm-table-strategy"` // 设备模型上报数据表后缀策略 1，年 2，月 3，日 其它不追加后缀
	dms              bool   // 缓存设备模型判断测试
}

func getTimeNowFormatSuffix(strategy uint) string {
	// 年
	if strategy == 1 {
		return toolkit.GetTimeNowFormat("2006")
	}
	// 月
	if strategy == 2 {
		return toolkit.GetTimeNowFormat("200601")
	}
	// 日
	if strategy == 3 {
		return toolkit.GetTimeNowFormat("20060102")
	}
	return ""
}

func (t *TimeSeriesDb) GetDudTableSuffix() string {
	// 设备上报数据表后缀
	if t.dts == 0 {
		strategy := t.DudTableStrategy
		if strategy > 3 || strategy < 1 {
			strategy = 1
		}
		t.dts = strategy
	}
	return getTimeNowFormatSuffix(t.dts)
}

func (t *TimeSeriesDb) GetDdmTableName(tableName string) string {
	// 设备模型上报数据表后缀
	if t.dms {
		return tableName
	}
	suffix := getTimeNowFormatSuffix(t.DdmTableStrategy)
	if suffix == "" {
		t.dms = true
		return tableName
	}
	return fmt.Sprintf("%s_%s", tableName, suffix)
}
