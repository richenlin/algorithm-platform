package database

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteProvider SQLite 数据库提供者实现
type SQLiteProvider struct {
	dbPath string
	db     *gorm.DB
}

// NewSQLiteProvider 创建 SQLite 数据库提供者
func NewSQLiteProvider(dbPath string) *SQLiteProvider {
	return &SQLiteProvider{
		dbPath: dbPath,
	}
}

// Open 打开 SQLite 数据库连接
func (p *SQLiteProvider) Open() (*gorm.DB, error) {
	// 确保数据目录存在
	dataDir := filepath.Dir(p.dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 打开数据库
	db, err := gorm.Open(sqlite.Dialector{
		DSN: p.dbPath,
	}, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	p.db = db
	return db, nil
}

// Configure 配置 SQLite 数据库连接参数
func (p *SQLiteProvider) Configure(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// SQLite 推荐配置：单连接以避免并发问题
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	return nil
}

// Close 关闭 SQLite 数据库连接
func (p *SQLiteProvider) Close() error {
	if p.db == nil {
		return nil
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Close()
}

// Name 返回数据库提供者名称
func (p *SQLiteProvider) Name() string {
	return "SQLite"
}

// Ping 测试 SQLite 数据库连接
func (p *SQLiteProvider) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Ping()
}
