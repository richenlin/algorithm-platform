package database

import (
	"gorm.io/gorm"
)

// DBProvider 定义数据库提供者接口
// 支持不同的数据库实现(SQLite, PostgreSQL等)
type DBProvider interface {
	// Open 打开数据库连接
	Open() (*gorm.DB, error)

	// Configure 配置数据库连接参数
	Configure(db *gorm.DB) error

	// Close 关闭数据库连接
	Close() error

	// Name 返回数据库提供者名称
	Name() string

	// Ping 测试数据库连接
	Ping() error
}
