package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgreSQLProvider PostgreSQL 数据库提供者实现
type PostgreSQLProvider struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
	sslMode  string
	timezone string
	db       *gorm.DB
}

// PostgreSQLConfig PostgreSQL 配置
type PostgreSQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // disable, require, verify-ca, verify-full
	Timezone string
}

// NewPostgreSQLProvider 创建 PostgreSQL 数据库提供者
func NewPostgreSQLProvider(cfg PostgreSQLConfig) *PostgreSQLProvider {
	// 设置默认值
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Shanghai"
	}

	return &PostgreSQLProvider{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		dbname:   cfg.DBName,
		sslMode:  cfg.SSLMode,
		timezone: cfg.Timezone,
	}
}

// Open 打开 PostgreSQL 数据库连接
func (p *PostgreSQLProvider) Open() (*gorm.DB, error) {
	// 构建 DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		p.host, p.user, p.password, p.dbname, p.port, p.sslMode, p.timezone)

	// 打开数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	p.db = db
	return db, nil
}

// Configure 配置 PostgreSQL 数据库连接参数
func (p *PostgreSQLProvider) Configure(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// PostgreSQL 推荐配置
	sqlDB.SetMaxOpenConns(25)   // 最大打开连接数
	sqlDB.SetMaxIdleConns(5)    // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(0) // 连接最大生存时间 (0 表示永不过期)

	return nil
}

// Close 关闭 PostgreSQL 数据库连接
func (p *PostgreSQLProvider) Close() error {
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
func (p *PostgreSQLProvider) Name() string {
	return "PostgreSQL"
}

// Ping 测试 PostgreSQL 数据库连接
func (p *PostgreSQLProvider) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Ping()
}
