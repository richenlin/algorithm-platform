# 算法管理平台解决方案 (SOLUTION.md) v7

## 1. 方案概述

本方案在 v6 (Dashboard 版) 基础上进行了关键的**稳定性与运维增强**补充，涵盖资源配额、日志系统、超时控制及安全加固，以满足生产环境要求。

核心特性：
1.  **Dashboard 管理面板**: 全生命周期管理（算法/数据/任务）。
2.  **双协议接入**: gRPC + RESTful。
3.  **存储解耦**: MinIO + Smart Runner。
4.  **计算去重**: Redis 缓存指纹。
5.  **稳定性增强**: 资源限制、超时熔断、日志审计、Matlab/C++ 优化。

## 2. 系统架构设计

### 2.1. 前端架构 (Dashboard)
(同 v6 设计)

### 2.2. 后端架构 (Core Platform)
(同 v6 设计)

## 3. 功能模块详细设计

### 3.1. 算法管理与运行 (增强版)

#### 3.1.1. 资源配额 (Resource Quotas)
为了防止某个算法耗尽宿主机资源，支持在算法定义时配置配额。
*   **配置项**:
    *   `cpu_limit`: 最大 CPU 核数 (e.g., "0.5", "2").
    *   `memory_limit`: 最大内存 (e.g., "512MB", "4GB").
*   **实现**: 平台调用 Docker API 时传入 `NanoCPUs` 和 `Memory` 参数。

#### 3.1.2. 任务超时与熔断 (Timeout)
*   **配置项**: `timeout_seconds` (默认 600s).
*   **机制**:
    *   Go 平台层使用 `context.WithTimeout` 控制。
    *   若超时，平台主动向 Docker 发送 `SIGKILL`，并标记任务状态为 `timeout`。

#### 3.1.3. 日志收集 (Logging)
*   **采集**: `algo-runner` 捕获子进程的 `stdout` 和 `stderr`。
*   **存储**: 算法运行结束后，日志被打包为 `run.log` 并上传至 MinIO Outputs 目录。
*   **展示**: Dashboard 任务详情页增加“查看日志”功能，从 MinIO 读取日志内容。

### 3.2. 镜像构建优化 (Matlab/C++)
*   **Matlab**: 针对 MCR (Matlab Compiler Runtime) 基础镜像过大 (4GB+) 问题，采用**节点预热**策略，在部署时预先拉取基础镜像。
*   **C++**: 强制使用**多阶段构建 (Multi-stage Build)**。
    ```dockerfile
    # Stage 1: Build
    FROM gcc:latest AS builder
    COPY . /src
    RUN g++ -o myapp main.cpp
    
    # Stage 2: Runtime
    FROM debian:slim
    COPY --from=builder /src/myapp /app/myapp
    ```
    有效将镜像体积从 >1GB 压缩至 <100MB。

### 3.3. 数据管理模块
(同 v6 设计)

### 3.4. 运维与安全 (Ops & Security)
*   **定期清理**: 启用 Cron Job，每日执行 `docker system prune` 清理悬空镜像和 Exited 容器。
*   **网络安全**: 算法容器启动时默认使用受限网络，仅允许访问 MinIO 和必要的内网服务，禁止公网访问。
*   **非 Root 运行**: 强制容器内使用非 Root 用户 (UID 1000)，降低容器逃逸风险。

## 4. 接口定义 (Protobuf)
(同 v6 设计，字段增加 `resource_config`)

## 5. 开发计划
1.  **阶段一**: 核心调度引擎 (gRPC/Docker) + 资源限制/超时功能。
2.  **阶段二**: MinIO 集成 (Runner/Logging)。
3.  **阶段三**: Dashboard (算法/数据/任务管理 UI)。
4.  **阶段四**: 安全加固与构建优化 (Matlab/C++)。

## 6. 总结
本方案 (v7) 是一个生产就绪的架构。通过增加资源配额、日志收集和安全限制，系统不仅“能用”，而且“稳健”。Dashboard 的加入使得非技术人员也能轻松管理复杂的算法资产。
