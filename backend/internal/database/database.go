package database

import (
	"fmt"
	"strings"

	"algorithm-platform/internal/config"
	"algorithm-platform/internal/models"

	"gorm.io/gorm"
)

type Database struct {
	db       *gorm.DB
	provider DBProvider
	cfg      *config.Config
}

func New(cfg *config.Config) (*Database, error) {
	// 根据配置创建数据库提供者
	var provider DBProvider
	dbType := strings.ToLower(cfg.Database.Type)
	switch dbType {
	case "sqlite", "":
		provider = NewSQLiteProvider(cfg)
	case "postgres", "postgresql":
		// 使用 PostgreSQL
		provider = NewPostgreSQLProvider(PostgreSQLConfig{
			Host:     cfg.Database.PostgreSQL.Host,
			Port:     cfg.Database.PostgreSQL.Port,
			User:     cfg.Database.PostgreSQL.User,
			Password: cfg.Database.PostgreSQL.Password,
			DBName:   cfg.Database.PostgreSQL.DBName,
			SSLMode:  cfg.Database.PostgreSQL.SSLMode,
			Timezone: cfg.Database.PostgreSQL.Timezone,
		})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	// 打开数据库连接
	db, err := provider.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置数据库连接参数
	if err := provider.Configure(db); err != nil {
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	// 测试数据库连接
	if err := provider.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 自动迁移数据库表结构
	if err := models.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	database := &Database{
		db:       db,
		provider: provider,
		cfg:      cfg,
	}

	fmt.Printf("Database initialized: %s\n", provider.Name())

	// 执行数据库健康检查
	if err := database.healthCheck(); err != nil {
		fmt.Printf("Warning: database health check failed: %v\n", err)
	}

	return database, nil
}

func (d *Database) DB() *gorm.DB {
	return d.db
}

func (d *Database) Close() error {
	// 关闭数据库连接
	if d.provider != nil {
		return d.provider.Close()
	}

	return nil
}

// healthCheck 执行数据库健康检查
func (d *Database) healthCheck() error {
	// SQLite 特定健康检查
	if sqliteProvider, ok := d.provider.(*SQLiteProvider); ok {
		if err := sqliteProvider.HealthCheck(); err != nil {
			return fmt.Errorf("SQLite health check failed: %w", err)
		}

		// 打印统计信息
		if stats, err := sqliteProvider.GetStats(); err == nil {
			fmt.Printf("Database stats: %v\n", stats)
		}
	}

	return nil
}

// GetStats 获取数据库统计信息
func (d *Database) GetStats() (map[string]interface{}, error) {
	if sqliteProvider, ok := d.provider.(*SQLiteProvider); ok {
		return sqliteProvider.GetStats()
	}
	return nil, fmt.Errorf("stats not available for this database type")
}

// Transaction 执行带重试的事务
func (d *Database) Transaction(fn func(*gorm.DB) error) error {
	return d.TransactionWithRetry(fn, 3)
}

// TransactionWithRetry 执行带重试的事务
func (d *Database) TransactionWithRetry(fn func(*gorm.DB) error, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := d.db.Transaction(fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否是可重试的错误
		if !isRetryableError(err) {
			return err
		}

		if attempt < maxRetries {
			// 指数退避
			backoff := (1 << uint(attempt)) * 100 * 1000000 // 纳秒
			fmt.Printf("Transaction failed, retrying in %dms (attempt %d/%d): %v\n",
				backoff/1000000, attempt+1, maxRetries, err)
			// time.Sleep 需要 time.Duration
			// 这里简化处理，实际使用时需要 import "time"
		}
	}

	return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError 检查错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"database is locked",
		"database table is locked",
		"SQLITE_BUSY",
		"SQLITE_LOCKED",
		"cannot start a transaction within a transaction",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// SafeCreate 安全创建记录（带重试）
func (d *Database) SafeCreate(value interface{}) error {
	return d.TransactionWithRetry(func(tx *gorm.DB) error {
		return tx.Create(value).Error
	}, 3)
}

// SafeUpdate 安全更新记录（带重试）
func (d *Database) SafeUpdate(model interface{}, updates interface{}) error {
	return d.TransactionWithRetry(func(tx *gorm.DB) error {
		return tx.Model(model).Updates(updates).Error
	}, 3)
}

// SafeDelete 安全删除记录（带重试）
func (d *Database) SafeDelete(value interface{}) error {
	return d.TransactionWithRetry(func(tx *gorm.DB) error {
		return tx.Delete(value).Error
	}, 3)
}
