package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations 执行数据库迁移
// 自动检测当前版本并应用所有未执行的迁移
func RunMigrations(db *sql.DB, logger *zap.Logger) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("加载迁移文件失败: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("创建迁移驱动失败: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("初始化迁移实例失败: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("执行迁移失败: %w", err)
	}

	version, dirty, _ := m.Version()
	if dirty {
		logger.Warn("数据库迁移处于 dirty 状态", zap.Uint("version", version))
	} else {
		logger.Info("数据库迁移完成", zap.Uint("version", version))
	}

	return nil
}
