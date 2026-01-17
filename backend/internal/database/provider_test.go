package database

import (
	"path/filepath"
	"testing"

	"algorithm-platform/internal/config"
	"algorithm-platform/internal/models"
)

func TestSQLiteProvider(t *testing.T) {
	// 使用临时文件而不是内存数据库（内存数据库不支持 WAL）
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// 创建测试配置（不包含 MinIO，避免连接错误）
	testCfg := &config.Config{
		Database: config.DatabaseConfig{
			Type: "sqlite",
			SQLite: config.SQLiteConfig{
				Path:                     dbPath,
				WALCheckpointIntervalStr: "30s",
			},
		},
		MinIO: config.MinIOConfig{
			Endpoint:        "test:9000",
			Bucket:          "test",
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		},
	}

	// 创建 SQLite 提供者
	provider := NewSQLiteProvider(testCfg)

	// 测试打开数据库
	db, err := provider.Open()
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}

	// 测试配置数据库
	err = provider.Configure(db)
	if err != nil {
		t.Fatalf("Failed to configure SQLite database: %v", err)
	}

	// 迁移表结构（模拟 Database.New 的行为）
	err = models.AutoMigrate(db)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 执行迁移后操作
	err = provider.PostMigrate()
	if err != nil {
		t.Fatalf("Failed to execute post-migration: %v", err)
	}

	// 测试 Ping
	err = provider.Ping()
	if err != nil {
		t.Fatalf("Failed to ping SQLite database: %v", err)
	}

	// 验证名称
	if provider.Name() != "SQLite" {
		t.Errorf("Expected provider name 'SQLite', got '%s'", provider.Name())
	}

	// 测试健康检查
	err = provider.HealthCheck()
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// 测试获取统计信息
	stats, err := provider.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats["page_count"] == nil {
		t.Error("Expected page_count in stats")
	}

	// 测试关闭
	err = provider.Close()
	if err != nil {
		t.Fatalf("Failed to close SQLite database: %v", err)
	}
}

func TestDatabaseInitialization(t *testing.T) {
	// 创建测试配置
	_ = &config.Config{
		Server: config.ServerConfig{
			GRPCPort: 9090,
			HTTPPort: 8080,
		},
		MinIO: config.MinIOConfig{
			Endpoint:         "localhost:9000",
			ExternalEndpoint: "localhost:9000",
			AccessKeyID:      "minioadmin",
			SecretAccessKey:  "minioadmin",
			Bucket:           "test-bucket",
			UseSSL:           false,
		},
		Database: config.DatabaseConfig{
			Type: "sqlite",
			SQLite: config.SQLiteConfig{
				Path: ":memory:",
			},
		},
	}

	// 注意：这个测试会失败如果 MinIO 不可用
	// 这是预期的，因为 Database.New() 会尝试连接 MinIO
	// 在实际环境中，可以跳过这个测试或使用 mock
	t.Run("SQLite", func(t *testing.T) {
		t.Skip("Skipping integration test - requires MinIO")
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		t.Skip("Skipping integration test - requires PostgreSQL and MinIO")
	})
}

func TestPostgreSQLProvider(t *testing.T) {
	t.Skip("Skipping PostgreSQL test - requires PostgreSQL server")

	// 如果有 PostgreSQL 服务器可用，可以取消注释以下代码
	/*
		provider := NewPostgreSQLProvider(PostgreSQLConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			DBName:   "test_db",
			SSLMode:  "disable",
			Timezone: "Asia/Shanghai",
		})

		db, err := provider.Open()
		if err != nil {
			t.Fatalf("Failed to open PostgreSQL database: %v", err)
		}

		err = provider.Configure(db)
		if err != nil {
			t.Fatalf("Failed to configure PostgreSQL database: %v", err)
		}

		err = provider.Ping()
		if err != nil {
			t.Fatalf("Failed to ping PostgreSQL database: %v", err)
		}

		if provider.Name() != "PostgreSQL" {
			t.Errorf("Expected provider name 'PostgreSQL', got '%s'", provider.Name())
		}

		err = provider.Close()
		if err != nil {
			t.Fatalf("Failed to close PostgreSQL database: %v", err)
		}
	*/
}
