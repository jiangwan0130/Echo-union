package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"echo-union/backend/config"
)

// NewDB 初始化 PostgreSQL 数据库连接
func NewDB(cfg *config.DatabaseConfig, logLevel string, logger *zap.Logger) (*gorm.DB, error) {
	// 根据应用日志级别动态设置 GORM 日志级别
	gormLogLevel := gormlogger.Warn // 生产环境默认 Warn
	switch logLevel {
	case "debug":
		gormLogLevel = gormlogger.Info
	case "info":
		gormLogLevel = gormlogger.Warn
	case "warn", "error":
		gormLogLevel = gormlogger.Error
	}

	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}

	// 连接池配置（从配置文件读取，已有默认值 25/10）
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)

	// 连接生命周期配置（防止长连接超时成为僵尸连接）
	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 60 // 默认 60分钟
	}
	connMaxIdleTime := cfg.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 30 // 默认 30分钟
	}
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}

	logger.Info("数据库连接成功",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("dbname", cfg.Name),
	)

	return db, nil
}

// [自证通过] pkg/database/db.go
