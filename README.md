# 算法管理平台

一个基于 Go + Docker 的算法管理平台，支持 Python、Matlab、C++ 等多语言算法的统一调度、版本控制、在线开发和缓存优化。

## 架构特点

- **双协议接入**: gRPC (9090) + RESTful (8080)
- **容器化调度**: 基于 Docker 的算法容器管理
- **资源限制**: CPU/Memory 配额控制
- **缓存优化**: Redis 去重缓存
- **存储解耦**: MinIO 对象存储
- **版本控制**: 算法版本回滚

## 快速开始

### 使用 Docker Compose（推荐）

```bash
docker-compose -f deploy/docker-compose.yml up -d
```

### 本地开发

1. **配置文件**

使用 Makefile 快速初始化配置：
```bash
make config-init
```

或手动复制配置示例文件：
```bash
cp backend/config/config.example.yaml backend/config/config.yaml
```

根据本地环境修改 `backend/config/config.yaml`：
- 配置 MinIO 连接信息
- 配置 Redis 连接信息
- 配置服务端口
- 配置数据库类型（SQLite 或 PostgreSQL）

2. **启动依赖服务**

确保 MinIO 和 Redis 服务正在运行：
```bash
# 使用 Docker 启动 MinIO
docker run -d -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  minio/minio server /data --console-address ":9001"

# 使用 Docker 启动 Redis
docker run -d -p 6379:6379 redis:alpine

# （可选）使用 Docker 启动 PostgreSQL（生产环境推荐）
docker run -d -p 5432:5432 \
  -e "POSTGRES_USER=postgres" \
  -e "POSTGRES_PASSWORD=postgres" \
  -e "POSTGRES_DB=algorithm_platform" \
  postgres:16-alpine
```

**数据库选择**：
- **SQLite**（默认）：零配置，适合开发环境
- **PostgreSQL**：高性能，适合生产环境

在 `config.yaml` 中配置数据库：
```yaml
database:
  type: "sqlite"  # 或 "postgres"
  sqlite_path: "./data/algorithm-platform.db"
  postgresql:
    host: "localhost"
    port: 5432
    user: "postgres"
    password: "postgres"
    dbname: "algorithm_platform"
```

详见：[数据库配置文档](backend/internal/database/README.md)

3. **运行服务**

使用 Makefile（推荐）：
```bash
# 验证配置并运行（本地开发模式）
make dev

# 或者只运行服务
make run-local

# 查看所有可用命令
make help
```

或直接使用 go run：
```bash
# 本地开发模式（使用 localhost:9000 连接 MinIO）
LOCAL_MODE=true go run ./backend/cmd/server/main.go

# 或直接运行（使用配置文件中的设置）
go run ./backend/cmd/server/main.go
```

## API 接口

### 执行算法

**gRPC**: `AlgorithmService.ExecuteAlgorithm`

**RESTful**:
```bash
POST http://localhost:8080/api/v1/algorithms/{algorithm_id}/execute
Content-Type: application/json

{
  "mode": "async",
  "params": {"threshold": 0.5},
  "input_source": {
    "type": "minio_url",
    "url": "minio/bucket/data.csv"
  },
  "webhook_url": "http://callback.url/webhook"
}
```

## 目录结构

```
.
├── backend/              # 后端代码
│   ├── api/v1/proto/     # gRPC 生成代码
│   ├── cmd/              # 程序入口
│   │   ├── server/       # 主服务
│   │   └── config-validator/ # 配置验证工具
│   ├── config/           # 配置文件目录
│   │   ├── config.yaml           # 主配置文件
│   │   └── config.example.yaml   # 配置示例
│   ├── internal/         # 内部包
│   │   ├── config/       # 配置管理
│   │   ├── database/     # 数据库管理
│   │   ├── models/       # 数据库模型
│   │   ├── scheduler/    # Docker 调度器
│   │   ├── server/       # gRPC/HTTP Server
│   │   └── service/      # 业务逻辑
│   ├── pkg/              # 公共包
│   │   ├── cache/        # Redis 缓存
│   │   ├── docker/       # Docker 客户端
│   │   └── storage/      # MinIO 存储
│   ├── proto/            # Protobuf 定义
│   ├── runner/           # Smart Runner
│   ├── data/             # 数据文件
│   ├── go.mod            # Go 依赖管理
│   └── go.sum
├── deploy/               # 部署配置
│   ├── Dockerfile        # 主服务镜像
│   ├── Dockerfile.runner # Runner 镜像
│   ├── Dockerfile.python # Python 算法镜像
│   ├── Dockerfile.cpp    # C++ 算法镜像
│   └── docker-compose.yml # 容器编排
├── doc/                  # 文档
│   ├── SOLUTION.md       # 解决方案文档
│   └── TASK.md           # 任务说明
├── frontend/             # 前端代码
│   └── ...
├── test/                 # 测试
│   ├── scripts/          # 测试脚本
│   └── test-api.sh       # API 测试
├── Makefile              # 构建和开发命令
├── README.md             # 项目说明
└── progress.txt          # 进度记录
```

## 开发

### 常用命令（Makefile）

```bash
# 查看所有可用命令
make help

# 初始化配置文件
make config-init

# 验证配置
make config-validate

# 运行服务（开发模式）
make run-local

# 验证配置并运行
make dev

# 构建二进制文件
make build

# 运行测试
make test

# 生成 Protobuf 代码
make proto

# 整理依赖
make tidy
```

### 手动运行

```bash
# 安装依赖
cd backend && go mod download

# 配置文件
cp backend/config/config.example.yaml backend/config/config.yaml
# 编辑 backend/config/config.yaml 设置本地环境配置

# 运行服务（本地模式）
LOCAL_MODE=true go run ./backend/cmd/server/main.go

# 生成 Protobuf 代码
cd backend && buf generate

# 运行测试
cd backend && go test ./...
```

## 配置说明

配置文件位于 `backend/config/config.yaml`，主要配置项：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.grpc_port` | gRPC 服务端口 | 9090 |
| `server.http_port` | HTTP/REST 服务端口 | 8080 |
| `minio.endpoint` | MinIO 内部访问地址 | minio:9000 |
| `minio.external_endpoint` | MinIO 外部访问地址 | localhost:9000 |
| `minio.access_key_id` | MinIO 访问密钥 | minioadmin |
| `minio.secret_access_key` | MinIO 密钥 | minioadmin |
| `redis.addr` | Redis 服务地址 | localhost:6379 |

**环境变量覆盖：**
- `LOCAL_MODE=true`: 强制使用 localhost:9000 连接 MinIO（适用于本地开发）

## 部署

```bash
# 构建镜像
docker build -t algorithm-platform -f deploy/Dockerfile .

# 使用 docker-compose 部署
docker-compose -f deploy/docker-compose.yml up -d
```

## License

MIT
