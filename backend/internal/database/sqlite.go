package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteProvider SQLite 数据库提供者实现
type SQLiteProvider struct {
	dbPath                string
	db                    *gorm.DB
	walCheckpointInterval time.Duration
	stopCheckpoint        chan struct{}
}

// SQLiteConfig SQLite 配置选项
type SQLiteConfig struct {
	Path                  string
	WALCheckpointInterval time.Duration // WAL checkpoint 间隔，默认 30 秒
}

// NewSQLiteProvider 创建 SQLite 数据库提供者
func NewSQLiteProvider(dbPath string) *SQLiteProvider {
	return NewSQLiteProviderWithConfig(SQLiteConfig{
		Path:                  dbPath,
		WALCheckpointInterval: 30 * time.Second,
	})
}

// NewSQLiteProviderWithConfig 使用配置创建 SQLite 数据库提供者
func NewSQLiteProviderWithConfig(cfg SQLiteConfig) *SQLiteProvider {
	if cfg.WALCheckpointInterval == 0 {
		cfg.WALCheckpointInterval = 30 * time.Second
	}
	return &SQLiteProvider{
		dbPath:                cfg.Path,
		walCheckpointInterval: cfg.WALCheckpointInterval,
		stopCheckpoint:        make(chan struct{}),
	}
}

// Open 打开 SQLite 数据库连接
func (p *SQLiteProvider) Open() (*gorm.DB, error) {
	// 确保数据目录存在
	dataDir := filepath.Dir(p.dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 打开数据库，启用共享缓存和扩展结果代码
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=FULL&_busy_timeout=5000&_foreign_keys=ON", p.dbPath)

	db, err := gorm.Open(sqlite.Dialector{
		DSN: dsn,
	}, &gorm.Config{
		// 预处理语句缓存
		PrepareStmt: true,
		// 不自动 Ping，我们手动处理
		DisableAutomaticPing: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	p.db = db

	// 初始化数据库优化设置
	if err := p.optimizeDatabase(); err != nil {
		return nil, fmt.Errorf("failed to optimize database: %w", err)
	}

	// 启动 WAL checkpoint 定时任务
	go p.walCheckpointWorker()

	return db, nil
}

// optimizeDatabase 优化数据库设置
func (p *SQLiteProvider) optimizeDatabase() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}

	pragmas := []struct {
		name  string
		value string
		desc  string
	}{
		// WAL 模式：提供更好的并发性和崩溃恢复
		{"journal_mode", "WAL", "启用 Write-Ahead Logging"},

		// FULL 同步：确保数据在系统崩溃时不会丢失
		{"synchronous", "FULL", "最高数据安全级别"},

		// 启用外键约束
		{"foreign_keys", "ON", "强制外键完整性"},

		// 设置缓存大小（2000 pages ≈ 8MB，假设 page_size = 4096）
		{"cache_size", "-8000", "8MB 缓存"},

		// 设置临时存储在内存中
		{"temp_store", "MEMORY", "临时数据使用内存"},

		// 设置 mmap 大小（30MB），提高读取性能
		{"mmap_size", "30000000", "使用内存映射 I/O"},

		// 自动清理
		{"auto_vacuum", "INCREMENTAL", "启用增量自动清理"},
	}

	for _, pragma := range pragmas {
		result := sqlDB.QueryRow(fmt.Sprintf("PRAGMA %s = %s", pragma.name, pragma.value))
		var returnValue string
		if err := result.Scan(&returnValue); err != nil {
			// 某些 PRAGMA 不返回值，忽略错误
			continue
		}
		fmt.Printf("SQLite PRAGMA %s = %s (%s)\n", pragma.name, returnValue, pragma.desc)
	}

	// 验证 WAL 模式已启用
	var journalMode string
	if err := sqlDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("failed to verify journal mode: %w", err)
	}
	if journalMode != "wal" {
		return fmt.Errorf("WAL mode not enabled, got: %s", journalMode)
	}

	fmt.Println("SQLite database optimized with WAL mode and safety settings")
	return nil
}

// Configure 配置 SQLite 数据库连接参数
func (p *SQLiteProvider) Configure(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// WAL 模式下可以支持更多的并发读取
	// 但写入仍然是串行的，所以限制写入连接数
	sqlDB.SetMaxOpenConns(5)    // 允许多个读连接
	sqlDB.SetMaxIdleConns(2)    // 保持空闲连接
	sqlDB.SetConnMaxLifetime(0) // 连接不过期

	return nil
}

// walCheckpointWorker 定期执行 WAL checkpoint
func (p *SQLiteProvider) walCheckpointWorker() {
	ticker := time.NewTicker(p.walCheckpointInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.checkpoint(); err != nil {
				fmt.Printf("WAL checkpoint failed: %v\n", err)
			}
		case <-p.stopCheckpoint:
			// 退出前执行最后一次 checkpoint
			if err := p.checkpoint(); err != nil {
				fmt.Printf("Final WAL checkpoint failed: %v\n", err)
			}
			return
		}
	}
}

// checkpoint 执行 WAL checkpoint
func (p *SQLiteProvider) checkpoint() error {
	if p.db == nil {
		return nil
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}

	// PRAGMA wal_checkpoint(TRUNCATE) 会将 WAL 文件内容合并到主数据库并截断 WAL
	_, err = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("checkpoint failed: %w", err)
	}

	return nil
}

// Close 关闭 SQLite 数据库连接
func (p *SQLiteProvider) Close() error {
	// 停止 checkpoint worker
	close(p.stopCheckpoint)

	// 等待一小段时间让 checkpoint 完成
	time.Sleep(100 * time.Millisecond)

	if p.db == nil {
		return nil
	}

	// 最后一次 checkpoint 确保数据持久化
	if err := p.checkpoint(); err != nil {
		fmt.Printf("Warning: final checkpoint failed: %v\n", err)
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 执行 PRAGMA optimize
	if _, err := sqlDB.Exec("PRAGMA optimize"); err != nil {
		fmt.Printf("Warning: PRAGMA optimize failed: %v\n", err)
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

// HealthCheck 执行健康检查
func (p *SQLiteProvider) HealthCheck() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 检查数据库完整性
	var integrityCheck string
	err = sqlDB.QueryRow("PRAGMA integrity_check").Scan(&integrityCheck)
	if err != nil {
		return fmt.Errorf("integrity check query failed: %w", err)
	}

	if integrityCheck != "ok" {
		return fmt.Errorf("database integrity check failed: %s", integrityCheck)
	}

	// 检查 WAL 文件大小
	var walPages int
	err = sqlDB.QueryRow("PRAGMA wal_autocheckpoint").Scan(&walPages)
	if err == nil {
		fmt.Printf("WAL autocheckpoint: %d pages\n", walPages)
	}

	return nil
}

// GetStats 获取数据库统计信息
func (p *SQLiteProvider) GetStats() (map[string]interface{}, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})

	// 获取页面数量
	var pageCount int64
	if err := sqlDB.QueryRow("PRAGMA page_count").Scan(&pageCount); err == nil {
		stats["page_count"] = pageCount
	}

	// 获取页面大小
	var pageSize int64
	if err := sqlDB.QueryRow("PRAGMA page_size").Scan(&pageSize); err == nil {
		stats["page_size"] = pageSize
		stats["database_size_bytes"] = pageCount * pageSize
	}

	// 获取 freelist 页面数
	var freelistCount int64
	if err := sqlDB.QueryRow("PRAGMA freelist_count").Scan(&freelistCount); err == nil {
		stats["freelist_count"] = freelistCount
	}

	// 连接池统计
	dbStats := sqlDB.Stats()
	stats["open_connections"] = dbStats.OpenConnections
	stats["in_use"] = dbStats.InUse
	stats["idle"] = dbStats.Idle
	stats["wait_count"] = dbStats.WaitCount
	stats["wait_duration"] = dbStats.WaitDuration.String()

	return stats, nil
}

// Vacuum 执行数据库清理（释放未使用空间）
func (p *SQLiteProvider) Vacuum() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}

	fmt.Println("Running VACUUM on SQLite database...")
	if _, err := sqlDB.Exec("VACUUM"); err != nil {
		return fmt.Errorf("VACUUM failed: %w", err)
	}

	fmt.Println("VACUUM completed successfully")
	return nil
}

// Backup 备份数据库文件
func (p *SQLiteProvider) Backup(destPath string) error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 执行 checkpoint 确保数据完整性
	if err := p.checkpoint(); err != nil {
		return fmt.Errorf("checkpoint before backup failed: %w", err)
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}

	// 使用 SQLite backup API
	query := fmt.Sprintf("VACUUM INTO '%s'", destPath)
	if _, err := sqlDB.Exec(query); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Printf("Database backed up to: %s\n", destPath)
	return nil
}

// ExecuteWithRetry 执行带重试的数据库操作
func (p *SQLiteProvider) ExecuteWithRetry(fn func(*sql.DB) error, maxRetries int) error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		err := fn(sqlDB)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否是可重试的错误（如数据库锁定）
		if !isSQLiteBusyError(err) {
			return err
		}

		// 指数退避
		if i < maxRetries {
			backoff := time.Duration(1<<uint(i)) * 10 * time.Millisecond
			fmt.Printf("SQLite busy, retrying in %v (attempt %d/%d)...\n", backoff, i+1, maxRetries)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isSQLiteBusyError 检查是否是 SQLite 忙碌错误
func isSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "database is locked" ||
		errStr == "database table is locked" ||
		errStr == "SQLITE_BUSY"
}
