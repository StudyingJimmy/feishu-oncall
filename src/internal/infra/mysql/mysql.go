// Package mysql 业务库连接（MySQL 8.0，对应 scripts/ 里部署的 bokeoncall-mysql）。
//
// 只负责“连接池 + 建表入口”，不放任何 SQL 业务；业务查询写在各 repository 里。
package mysql

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"bokeoncall/internal/conf"
)

// Open 建立连接池。
func Open(cfg conf.MySQLConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Warn),
		NowFunc:                                  func() time.Time { return time.Now().UTC() },
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接池失败: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

// Migrate 建表/迁移入口：把 model 层定义的实体传进来即可，例如
//
//	mysql.Migrate(db, &entity.Ticket{}, &entity.TicketEvent{})
//
// 骨架阶段够用；正式环境建议换成迁移工具（atlas / goose / gormigrate）。
func Migrate(db *gorm.DB, models ...any) error {
	if len(models) == 0 {
		return nil
	}
	return db.AutoMigrate(models...)
}

// Ping 健康检查用。
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
