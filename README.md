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

```bash
docker-compose up -d
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
├── api/v1/proto/       # gRPC 生成代码
├── cmd/server/         # 主程序入口
├── internal/
│   ├── config/        # 配置管理
│   ├── models/        # 数据库模型
│   ├── scheduler/     # Docker 调度器
│   ├── server/        # gRPC/HTTP Server
│   └── service/      # 业务逻辑
├── pkg/
│   ├── cache/         # Redis 缓存
│   ├── docker/        # Docker 客户端
│   └── storage/       # MinIO 存储
├── runner/            # Smart Runner
├── proto/            # Protobuf 定义
└── docker-compose.yml # 部署配置
```

## 开发

```bash
# 安装依赖
go mod download

# 运行服务
go run cmd/server/main.go

# 生成 Protobuf 代码
buf generate

# 运行测试
go test ./...
```

## 部署

```bash
# 构建镜像
docker build -t algorithm-platform .

# 使用 docker-compose 部署
docker-compose up -d
```

## License

MIT
