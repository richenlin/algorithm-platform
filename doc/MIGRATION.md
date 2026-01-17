# 数据库迁移指南

本指南介绍如何在 SQLite 和 PostgreSQL 之间进行迁移。

## SQLite 迁移到 PostgreSQL

### 前提条件

1. 安装 PostgreSQL 数据库
2. 创建数据库和用户

```sql
-- 以 postgres 用户登录
CREATE DATABASE algorithm_platform;
CREATE USER algorithm_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE algorithm_platform TO algorithm_user;
```

### 步骤 1: 备份 SQLite 数据

项目自动将数据备份到 MinIO，确保最新备份存在：

```bash
# 检查 MinIO 中的备份
mc ls minio/algorithm-platform/database-backup/
```

### 步骤 2: 配置 PostgreSQL

编辑 `config/config.yaml`：

```yaml
database:
  type: "postgres"
  postgresql:
    host: "localhost"
    port: 5432
    user: "algorithm_user"
    password: "your_password"
    dbname: "algorithm_platform"
    sslmode: "disable"  # 生产环境建议使用 require
    timezone: "Asia/Shanghai"
```

### 步骤 3: 启动服务

```bash
cd backend
go run cmd/server/main.go
```

服务会自动：
1. 连接到 PostgreSQL
2. 创建所需的表结构（AutoMigrate）
3. 从 MinIO 恢复备份数据

### 步骤 4: 验证数据

检查数据是否正确迁移：

```bash
# 连接到 PostgreSQL
psql -U algorithm_user -d algorithm_platform

# 查看表
\dt

# 查看数据
SELECT COUNT(*) FROM algorithms;
SELECT COUNT(*) FROM jobs;
SELECT COUNT(*) FROM preset_data;
```

## PostgreSQL 迁移到 SQLite

### 步骤 1: 确保数据已备份

PostgreSQL 数据会定期备份到 MinIO。

### 步骤 2: 配置 SQLite

编辑 `config/config.yaml`：

```yaml
database:
  type: "sqlite"
  sqlite_path: "./data/algorithm-platform.db"
```

### 步骤 3: 启动服务

```bash
cd backend
go run cmd/server/main.go
```

数据将从 MinIO 自动恢复。

## 使用 Docker Compose 部署 PostgreSQL

在 `docker-compose.yml` 中添加 PostgreSQL 服务：

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: algorithm-postgres
    environment:
      POSTGRES_DB: algorithm_platform
      POSTGRES_USER: algorithm_user
      POSTGRES_PASSWORD: algorithm_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - algorithm-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U algorithm_user"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    # ... 其他配置
    depends_on:
      - postgres
    environment:
      # 数据库配置可以通过环境变量覆盖
      - DB_TYPE=postgres
      - DB_HOST=postgres
      - DB_PORT=5432

volumes:
  postgres_data:
```

然后更新 `config/config.yaml`：

```yaml
database:
  type: "postgres"
  postgresql:
    host: "postgres"  # Docker 服务名
    port: 5432
    user: "algorithm_user"
    password: "algorithm_pass"
    dbname: "algorithm_platform"
    sslmode: "disable"
    timezone: "Asia/Shanghai"
```

## 手动数据导出/导入

### 从 SQLite 导出

```bash
# 安装 sqlite3 命令行工具
sqlite3 data/algorithm-platform.db .dump > backup.sql
```

### 导入到 PostgreSQL

需要转换 SQL 语法（SQLite 和 PostgreSQL 语法有差异）：

```bash
# 1. 转换 SQLite dump 为 PostgreSQL 格式
# 推荐使用工具: pgloader

# 安装 pgloader (macOS)
brew install pgloader

# 执行迁移
pgloader sqlite://data/algorithm-platform.db postgresql://algorithm_user:password@localhost/algorithm_platform
```

### 使用 GORM 迁移工具

项目使用 GORM 的 AutoMigrate，表结构会自动创建。只需要迁移数据：

```go
// 示例迁移脚本
package main

import (
    "algorithm-platform/internal/models"
    "gorm.io/driver/sqlite"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // 连接 SQLite
    sqliteDB, _ := gorm.Open(sqlite.Open("data/algorithm-platform.db"), &gorm.Config{})
    
    // 连接 PostgreSQL
    postgresDB, _ := gorm.Open(postgres.Open("host=localhost user=algorithm_user password=password dbname=algorithm_platform"), &gorm.Config{})
    
    // 迁移 Algorithms
    var algorithms []models.Algorithm
    sqliteDB.Find(&algorithms)
    postgresDB.Create(&algorithms)
    
    // 迁移其他表...
}
```

## 性能对比

### SQLite
- **优点**: 零配置，文件型数据库，适合开发
- **缺点**: 并发写入性能有限
- **适用场景**: 开发环境、小规模部署、单机应用

### PostgreSQL
- **优点**: 高性能，支持高并发，功能丰富
- **缺点**: 需要单独部署和维护
- **适用场景**: 生产环境、高并发场景、分布式部署

## 常见问题

### Q: 可以在运行时切换数据库吗？

A: 不建议。需要停止服务、更新配置、重启服务。数据会从 MinIO 备份自动恢复。

### Q: 两种数据库可以同时使用吗？

A: 不支持。同一时间只能使用一种数据库类型。

### Q: 迁移会丢失数据吗？

A: 不会。所有数据都备份在 MinIO 中，迁移时会自动从备份恢复。

### Q: 如何回滚迁移？

A: 修改配置文件回到原来的数据库类型，重启服务即可。数据从 MinIO 恢复。

### Q: PostgreSQL 连接失败怎么办？

A: 检查：
1. PostgreSQL 是否运行: `pg_isready`
2. 防火墙是否开放端口 5432
3. `pg_hba.conf` 是否允许连接
4. 用户名密码是否正确

## 最佳实践

1. **生产环境使用 PostgreSQL**: 更好的性能和稳定性
2. **开发环境使用 SQLite**: 快速启动，无需额外服务
3. **定期检查备份**: 确保 MinIO 备份正常运行
4. **启用 SSL**: 生产环境设置 `sslmode: "require"`
5. **监控连接池**: 观察数据库连接数，适时调整配置
6. **测试迁移**: 先在测试环境验证迁移流程

## 自动备份

项目内置自动备份功能：
- 每 5 分钟自动备份到 MinIO
- 关闭服务时自动备份
- 备份包含所有表数据
- 保存为 JSON 格式

备份文件位置：
- `database-backup/latest.json` - 最新备份
- `database-backup/backup-YYYYMMDD-HHMMSS.json` - 历史备份
