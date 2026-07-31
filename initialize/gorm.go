package initialize

import (
	"apipig/config"
	"apipig/global"
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func Gorm() *gorm.DB {
	db, err := OpenDatabase(global.CONFIG.Database)
	if err != nil {
		global.LOG.Panic("database start error", zap.Error(err))
	}
	configureConnectionPool(global.CONFIG.Database, db)
	if err := initData(db, nil); err != nil {
		global.LOG.Panic("database initialization error", zap.Error(err))
	}
	return db
}

func OpenDatabase(dbCfg config.Database) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(dbCfg.DbType)) {
	case "", "sqlite":
		return gorm.Open(sqlite.Open(sqliteDSN(dbCfg)), gormConfig(dbCfg.LogMode))
	case "mysql":
		if dbCfg.Dbname == "" {
			return nil, fmt.Errorf("database configuration error: db-name cannot be empty")
		}
		dsn := dbCfg.Username + ":" + dbCfg.Password + "@tcp(" + dbCfg.Host + ":" + dbCfg.Port + ")/" + dbCfg.Dbname
		if dbCfg.Config != "" {
			dsn += "?" + strings.TrimPrefix(dbCfg.Config, "?")
		}
		return gorm.Open(mysql.Open(dsn), gormConfig(dbCfg.LogMode))
	case "postgresql", "postgres":
		if dbCfg.Dbname == "" {
			return nil, fmt.Errorf("database configuration error: db-name cannot be empty")
		}
		return openPostgres(dbCfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbCfg.DbType)
	}
}

func openPostgres(dbCfg config.Database) (*gorm.DB, error) {
	dsn := "host=" + dbCfg.Host + " user=" + dbCfg.Username + " password=" + dbCfg.Password +
		" dbname=" + dbCfg.Dbname + " port=" + dbCfg.Port + " " + dbCfg.Config
	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), gormConfig(dbCfg.LogMode))
}

func configureConnectionPool(dbCfg config.Database, db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
}

func sqliteDSN(dbCfg config.Database) string {
	dsn := strings.TrimSpace(dbCfg.SQLitePath)
	if dsn == "" {
		dsn = strings.TrimSpace(dbCfg.Dbname)
	}
	if dsn == "" {
		return "apipig.db?_busy_timeout=30000"
	}
	if strings.Contains(dsn, "?") || dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
		return dsn
	}
	if !strings.HasSuffix(strings.ToLower(dsn), ".db") {
		dsn += ".db"
	}
	return dsn
}

func gormConfig(logMode string) *gorm.Config {
	cfg := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "apipig_",
			SingularTable: true,
		},
	}
	switch logMode {
	case "silent", "Silent":
		cfg.Logger = logger.Default.LogMode(logger.Silent)
	case "error", "Error":
		cfg.Logger = logger.Default.LogMode(logger.Error)
	case "warn", "Warn":
		cfg.Logger = logger.Default.LogMode(logger.Warn)
	case "info", "Info":
		cfg.Logger = logger.Default.LogMode(logger.Info)
	default:
		cfg.Logger = logger.Discard
	}
	return cfg
}
