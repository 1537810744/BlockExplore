# BlockExplore 项目完全指南

> 写给 Go/Docker/Redis/Kafka/数据库小白的保姆级文档

---

## 目录

1. [项目是做什么的](#1-项目是做什么的)
2. [整体架构设计](#2-整体架构设计)
3. [技术栈详解](#3-技术栈详解)
4. [文件夹结构详解](#4-文件夹结构详解)
5. [数据流：一条区块数据是怎么从链上到你屏幕的](#5-数据流)
6. [每个文件是干什么的](#6-每个文件是干什么的)
7. [从 0 到 1 搭建过程](#7-从-0-到-1-搭建过程)
8. [如何启动项目](#8-如何启动项目)
9. [API 接口说明](#9-api-接口说明)
10. [常见问题](#10-常见问题)

---

## 1. 项目是做什么的

**一句话**：从以太坊、比特币、Solana 三条链的节点拉取区块和交易数据，存到自己的数据库里，然后通过 API 对外提供查询服务。

**类比理解**：
- 区块链就像一个公开的账本，每一页就是一个"区块"，每笔转账就是一笔"交易"
- 这个项目就是一个"抄写员"，不停地从区块链节点抄写最新的账页，存到自己的数据库
- 然后提供一个"查询窗口"（API），让别人可以查某个区块、某笔交易、某个地址

**核心功能**：
- 三条链的区块同步（ETH/BTC/SOL）
- 区块列表查询（分页）
- 区块详情查询
- 区块内交易列表
- 交易详情查询
- 地址交易历史
- 统一搜索（输入区块号/交易哈希/地址，自动识别类型）
- 代币价格查询和历史价格曲线

---

## 2. 整体架构设计

### 2.1 为什么用微服务？

想象一个餐厅：
- 如果只有一个厨师（单体应用），他要同时做菜、端盘子、收钱，很容易忙不过来
- 如果分成厨师、服务员、收银员（微服务），各司其职，效率高，某个岗位出问题不影响其他人

本项目的微服务分工：

```
┌─────────────────────────────────────────────────────────────┐
│                      整体架构图                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   区块链节点          消息队列           数据库              │
│   ┌─────────┐       ┌───────┐       ┌──────────┐           │
│   │ ETH节点  │──────►│       │       │          │           │
│   │ BTC节点  │──────►│ Kafka │──────►│PostgreSQL│           │
│   │ SOL节点  │──────►│       │       │          │           │
│   └─────────┘       └───────┘       └────┬─────┘           │
│        │                                  │                 │
│   Sync Worker                        ┌────▼─────┐           │
│   (同步工人)                         │  Redis   │           │
│                                      │  (缓存)  │           │
│                                      └────┬─────┘           │
│                                           │                 │
│   客户端        反向代理         API服务    │                 │
│   ┌──────┐     ┌───────┐     ┌────────┐  │                 │
│   │ 浏览器│────►│ Nginx │────►│ API    │◄─┘                 │
│   │ APP  │     │       │     │ 服务   │                     │
│   └──────┘     └───────┘     └────────┘                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 为什么用 Kafka（消息队列）？

**类比**：Kafka 就像一个"传菜窗口"

- 厨师（Sync Worker）做好菜（拉取到区块数据）后，放到传菜窗口（Kafka Topic）
- 服务员（Block Processor）从传菜窗口取菜，端给客人（写入数据库）
- 好处：厨师和服务员互不影响，厨师做快了菜会堆在窗口，服务员做快了会等新菜

**为什么不用直接写数据库？**
- 如果 Sync Worker 直接写数据库，一旦数据库慢了或挂了，同步就停了
- 用 Kafka 解耦后，即使数据库暂时不可用，数据也不会丢失（存在 Kafka 里）

### 2.3 为什么用 Redis（缓存）？

**类比**：Redis 就像一个"便签本"

- 查数据库就像去图书馆找书，要走过去、翻书架，比较慢
- Redis 就像把常用答案写在便签贴在桌上，一看就知道，非常快
- 但便签空间有限（内存贵），所以只缓存热门数据

**本项目缓存什么？**
- 最新的区块列表（每 30 秒过期）
- 区块详情（每 60 秒过期）
- 交易详情（每 60 秒过期）
- 代币价格（每 60 秒过期）

### 2.4 为什么用 Nginx？

**类比**：Nginx 就像"前台接待"

- 所有请求先到 Nginx，Nginx 再分发给对应的 API 服务
- 可以做限流（防止有人疯狂请求把服务器搞崩）
- 可以做负载均衡（如果有多个 API 服务实例，均匀分配请求）

---

## 3. 技术栈详解

### 3.1 Go 语言（Golang）

**是什么**：Google 开发的编程语言，特点是编译快、运行快、天生支持并发

**为什么选它**：
- 区块链同步需要同时监听三条链，Go 的 goroutine（轻量级线程）非常适合
- 编译成单个二进制文件，部署简单

**本项目用到的 Go 特性**：
- goroutine：并发执行多个同步任务
- channel：goroutine 之间的通信
- interface：定义通用接口

### 3.2 Gin 框架

**是什么**：Go 语言最流行的 Web 框架，用来处理 HTTP 请求

**类比**：Gin 就像一个"路由器"，根据 URL 把请求分发给对应的处理函数

```go
// 举例：当有人访问 /api/v1/blocks 时，调用 GetBlockList 函数
r.GET("/api/v1/blocks", handler.GetBlockList)
```

### 3.3 GORM

**是什么**：Go 语言的 ORM 库，让你用 Go 代码操作数据库，不用写 SQL

**类比**：GORM 就像一个"翻译官"，把 Go 代码翻译成 SQL 语句

```go
// Go 代码
db.Where("chain = ?", "eth").First(&block)

// GORM 翻译成 SQL
SELECT * FROM blocks WHERE chain = 'eth' LIMIT 1;
```

### 3.4 PostgreSQL

**是什么**：一个强大的开源关系型数据库

**为什么选它**：
- 支持复杂查询（JOIN、子查询）
- 数据可靠性高
- 支持 JSON 类型

### 3.5 Redis

**是什么**：一个内存数据库，速度极快，但数据不持久（断电就丢）

**为什么选它**：
- 读写速度比 PostgreSQL 快 100 倍
- 支持设置过期时间（缓存场景完美）

**数据结构**：
- String：存储简单的键值对（本项目主要用这个）
- List：列表
- Hash：哈希表
- Set：集合

### 3.6 Kafka

**是什么**：一个分布式消息队列，用于在服务之间传递消息

**核心概念**：
- **Topic（主题）**：消息的分类，本项目有 3 个 Topic：
  - `block.raw.eth`：以太坊区块数据
  - `block.raw.btc`：比特币区块数据
  - `block.raw.sol`：Solana 区块数据
- **Producer（生产者）**：发送消息的服务（Sync Worker）
- **Consumer（消费者）**：接收消息的服务（Block Processor）
- **Consumer Group（消费者组）**：多个消费者组成一组，共同消费一个 Topic 的消息

**消息的生命周期**：
```
Sync Worker 拉取区块 → 发送到 Kafka Topic → Block Processor 消费消息 → 写入数据库
```

### 3.7 Docker 和 Docker Compose

**是什么**：
- **Docker**：把应用和它的依赖打包成一个"容器"，在任何地方都能运行
- **Docker Compose**：一键启动多个容器的工具

**类比**：
- Docker 就像一个"集装箱"，把货物（应用）和说明书（依赖）打包在一起
- Docker Compose 就像一个"调度员"，按照清单（docker-compose.yaml）安排所有集装箱

### 3.8 Viper

**是什么**：Go 语言的配置管理库，用于读取配置文件和环境变量

**配置优先级**：环境变量 > .env 文件 > 默认值

### 3.9 Zap

**是什么**：Go 语言的高性能日志库

**日志级别**：
- Debug：调试信息（开发时用）
- Info：一般信息（运行时用）
- Warn：警告信息
- Error：错误信息
- Fatal：致命错误（会终止程序）

---

## 4. 文件夹结构详解

```
BlockExplore/
│
├── cmd/                            # 【入口目录】每个微服务的 main.go 放这里
│   ├── eth-sync-worker/
│   │   └── main.go                 # 以太坊同步 Worker 的入口
│   ├── btc-sync-worker/
│   │   └── main.go                 # 比特币同步 Worker 的入口
│   ├── sol-sync-worker/
│   │   └── main.go                 # Solana 同步 Worker 的入口
│   ├── block-processor/
│   │   └── main.go                 # 区块处理器的入口（Kafka → 数据库）
│   ├── query-api/
│   │   └── main.go                 # 查询 API 的入口（:8080）
│   ├── search-api/
│   │   └── main.go                 # 搜索 API 的入口（:8081）
│   └── price-api/
│       └── main.go                 # 价格 API 的入口（:8082）
│
├── internal/                       # 【核心代码目录】所有业务逻辑放这里
│   │                               # 命名 internal 意味着只能本项目使用，外部不能导入
│   ├── config/
│   │   └── config.go               # 配置管理（读取 .env 文件）
│   ├── model/
│   │   ├── block.go                # 区块数据模型（对应数据库 blocks 表）
│   │   ├── transaction.go          # 交易数据模型（对应 transactions 表）
│   │   ├── address.go              # 地址数据模型（对应 addresses 表）
│   │   └── price.go                # 价格数据模型（对应 price_history 表）
│   ├── client/
│   │   ├── eth_client.go           # 以太坊 RPC 客户端（与以太坊节点通信）
│   │   ├── btc_client.go           # 比特币 RPC 客户端
│   │   └── sol_client.go           # Solana RPC 客户端
│   ├── mq/
│   │   ├── producer.go             # Kafka 生产者（发送消息）
│   │   └── consumer.go             # Kafka 消费者（接收消息）
│   ├── repository/
│   │   ├── block_repo.go           # 区块数据访问层（数据库操作）
│   │   ├── tx_repo.go              # 交易数据访问层
│   │   ├── search_repo.go          # 搜索数据访问层
│   │   └── price_repo.go           # 价格数据访问层
│   ├── service/
│   │   ├── sync/
│   │   │   ├── eth_sync.go         # 以太坊同步业务逻辑
│   │   │   ├── btc_sync.go         # 比特币同步业务逻辑
│   │   │   └── sol_sync.go         # Solana 同步业务逻辑
│   │   ├── processor/
│   │   │   └── block_processor.go  # 区块处理业务逻辑
│   │   ├── query/
│   │   │   └── query_service.go    # 查询业务逻辑
│   │   └── price/
│   │       └── price_service.go    # 价格业务逻辑
│   ├── handler/
│   │   ├── block_handler.go        # 区块 HTTP 处理器
│   │   ├── tx_handler.go           # 交易 HTTP 处理器
│   │   ├── search_handler.go       # 搜索 HTTP 处理器
│   │   └── price_handler.go        # 价格 HTTP 处理器
│   ├── router/
│   │   └── router.go               # 路由注册（URL 和 Handler 的映射）
│   └── middleware/
│       ├── cors.go                 # 跨域中间件
│       ├── request_id.go           # 请求 ID 中间件
│       └── ratelimit.go            # 限流中间件
│
├── pkg/                            # 【公共工具库】可以被外部项目导入
│   ├── logger/
│   │   └── logger.go               # 日志封装
│   ├── cache/
│   │   └── redis.go                # Redis 缓存封装
│   └── errcode/
│       └── errcode.go              # 错误码定义
│
├── migrations/
│   └── 001_init.sql                # 数据库建表 SQL 脚本
│
├── docker-compose.yaml             # Docker Compose 编排文件
├── Dockerfile                      # Docker 镜像构建文件
├── nginx.conf                      # Nginx 配置文件
├── .env                            # 环境变量配置文件
├── .env.example                    # 配置文件模板
├── .gitignore                      # Git 忽略文件列表
├── go.mod                          # Go 模块定义（项目名和依赖）
├── go.sum                          # Go 依赖的校验和
├── README.md                       # 项目说明
└── HOW_IT_WORKS.md                 # 本文档
```

### 4.1 为什么这样分层？

这是 Go 项目最经典的分层方式，叫做**分层架构**：

```
Handler（处理 HTTP 请求）
    ↓ 调用
Service（业务逻辑）
    ↓ 调用
Repository（数据库操作）
    ↓ 调用
Model（数据模型）
```

**好处**：
- 每层只关心自己的事，改一层不影响其他层
- 方便测试（可以 mock 掉不需要的层）
- 代码复用（同一个 Service 可以被多个 Handler 调用）

---

## 5. 数据流

### 5.1 区块同步数据流（最核心的流程）

```
步骤1: Sync Worker 每隔 N 秒向区块链节点发送请求
       ETH: 每 12 秒（以太坊出块时间）
       BTC: 每 600 秒（比特币约 10 分钟一个块）
       SOL: 每 1 秒（Solana 出块极快）

步骤2: 节点返回最新区块数据（JSON 格式）

步骤3: Sync Worker 把数据封装成消息，发送到 Kafka

步骤4: Block Processor 从 Kafka 消费消息

步骤5: 解析区块数据，提取区块信息和交易列表

步骤6: 写入 PostgreSQL 数据库

步骤7: 更新 Redis 缓存
```

**具体代码流程**：

```
eth-sync-worker/main.go
    → service/sync/eth_sync.go: Run() 循环调用 sync()
    → client/eth_client.go: GetLatestBlockNumber() 获取最新区块高度
    → client/eth_client.go: GetBlockByNumber() 获取区块详情
    → mq/producer.go: Send() 发送到 Kafka

block-processor/main.go
    → mq/consumer.go: Consume() 从 Kafka 读取消息
    → service/processor/block_processor.go: Handle() 处理消息
    → repository/block_repo.go: CreateSingle() 写入区块
    → repository/tx_repo.go: Create() 批量写入交易
```

### 5.2 API 查询数据流

```
步骤1: 客户端发送 HTTP 请求
       GET /api/v1/blocks?chain=eth&page=1

步骤2: Nginx 接收请求，转发给 query-api

步骤3: Gin 路由匹配，调用对应的 Handler

步骤4: Handler 解析参数，调用 Service

步骤5: Service 先查 Redis 缓存
       - 命中缓存：直接返回
       - 未命中：查 PostgreSQL

步骤6: 查询结果写入 Redis 缓存（设置过期时间）

步骤7: 返回 JSON 响应给客户端
```

### 5.3 搜索数据流

```
用户输入: "0xabc..." 或 "25178170" 或 "1A1zP1..."

步骤1: 根据输入格式判断类型
       - 纯数字 → 区块高度
       - 0x 开头 42 字符 → 以太坊地址
       - 0x 开头 66 字符 → 以太坊交易哈希
       - 1/3/bc1 开头 → 比特币地址
       - 87-88 字符 Base58 → Solana 签名

步骤2: 根据类型查询对应的表

步骤3: 返回结果（包含类型标识）
```

---

## 6. 每个文件是干什么的

### 6.1 cmd/ 目录（入口文件）

每个 cmd 子目录都是一个独立的可执行程序。

**cmd/query-api/main.go** 的职责：
1. 加载配置
2. 初始化日志
3. 连接数据库
4. 连接 Redis
5. 创建 Repository → Service → Handler
6. 注册路由
7. 启动 HTTP 服务

**cmd/eth-sync-worker/main.go** 的职责：
1. 加载配置
2. 创建以太坊 RPC 客户端
3. 创建 Kafka 生产者
4. 创建同步 Worker
5. 启动循环同步

### 6.2 internal/config/（配置管理）

**config.go** 做了什么：
1. 使用 Viper 读取 .env 文件
2. 设置默认值
3. 解析到结构体供其他模块使用

### 6.3 internal/model/（数据模型）

**model/block.go** 定义了 Block 结构体，对应数据库的 blocks 表：

```go
type Block struct {
    ID          int64     // 主键 ID
    Chain       string    // 链标识: eth/btc/sol
    BlockNumber int64     // 区块高度
    BlockHash   string    // 区块哈希
    // ... 更多字段
}
```

GORM 会根据这个结构体自动生成 SQL 语句。

### 6.4 internal/client/（区块链客户端）

**client/eth_client.go** 做了什么：
1. 封装以太坊 JSON-RPC 请求
2. 解析十六进制数据（以太坊返回的数据是十六进制的）
3. 转换为我们的数据模型

**以太坊 JSON-RPC 是什么？**
- 以太坊节点提供的一种 API 协议
- 通过 HTTP POST 发送 JSON 格式的请求
- 例如获取最新区块高度：
  ```json
  {
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
  }
  ```

### 6.5 internal/mq/（消息队列）

**mq/producer.go** 做了什么：
1. 创建 Kafka Writer
2. 提供 Send() 方法发送消息

**mq/consumer.go** 做了什么：
1. 创建 Kafka Reader
2. 提供 Consume() 方法循环读取消息
3. 收到消息后调用业务处理函数

### 6.6 internal/repository/（数据访问层）

**repository/block_repo.go** 做了什么：
- Create()：批量插入区块
- CreateSingle()：插入单个区块
- GetByChainAndNumber()：根据链和区块号查询
- GetLatest()：获取最新区块
- GetList()：分页查询区块列表

### 6.7 internal/service/（业务逻辑层）

**service/sync/eth_sync.go** 做了什么：
1. 定时调用 RPC 获取最新区块
2. 将区块数据发送到 Kafka

**service/processor/block_processor.go** 做了什么：
1. 从 Kafka 消费消息
2. 解析区块和交易数据
3. 写入数据库

**service/query/query_service.go** 做了什么：
1. 提供区块、交易的查询功能
2. 实现 Redis 缓存逻辑

### 6.8 internal/handler/（HTTP 处理器）

**handler/block_handler.go** 做了什么：
1. 解析 HTTP 请求参数
2. 调用 Service 获取数据
3. 返回 JSON 响应

### 6.9 internal/router/（路由注册）

**router/router.go** 做了什么：
1. 定义 URL 和 Handler 的映射关系
2. 注册中间件

### 6.10 internal/middleware/（中间件）

**middleware/cors.go**：
- 添加跨域响应头，允许浏览器跨域访问

**middleware/request_id.go**：
- 为每个请求生成唯一 ID，用于日志追踪

**middleware/ratelimit.go**：
- 限制每个 IP 的请求频率，防止被攻击

---

## 7. 从 0 到 1 搭建过程

### 7.1 第一步：初始化 Go 模块

```bash
# 创建项目目录
mkdir BlockExplore
cd BlockExplore

# 初始化 Go 模块（生成 go.mod 文件）
go mod init blockexplore
```

go.mod 文件定义了项目名称和依赖。

### 7.2 第二步：创建目录结构

```bash
# 创建所有目录
mkdir -p cmd/{eth-sync-worker,btc-sync-worker,sol-sync-worker,block-processor,query-api,search-api,price-api}
mkdir -p internal/{config,model,repository,service/{sync,processor,query,price},handler,router,middleware,client,mq}
mkdir -p pkg/{logger,cache,errcode}
mkdir -p migrations
```

### 7.3 第三步：编写基础代码

按照依赖关系，从底层往上层写：

1. **model**（数据模型）→ 最底层，定义数据结构
2. **config**（配置）→ 读取配置
3. **pkg**（工具库）→ 日志、缓存、错误码
4. **client**（RPC 客户端）→ 与区块链通信
5. **mq**（消息队列）→ Kafka 生产者/消费者
6. **repository**（数据访问）→ 数据库操作
7. **service**（业务逻辑）→ 核心业务
8. **handler**（HTTP 处理）→ 接收请求
9. **router**（路由）→ URL 映射
10. **middleware**（中间件）→ 通用处理
11. **cmd**（入口）→ 启动服务

### 7.4 第四步：添加依赖

```bash
# 添加所有依赖并下载
go mod tidy
```

这个命令会：
1. 扫描所有 .go 文件中的 import
2. 自动添加缺失的依赖到 go.mod
3. 下载依赖到本地缓存
4. 生成 go.sum 文件（依赖的校验和）

### 7.5 第五步：编写 Docker 配置

1. **Dockerfile**：定义如何构建镜像
2. **docker-compose.yaml**：定义如何编排服务
3. **nginx.conf**：定义 Nginx 配置

### 7.6 第六步：编写数据库迁移

**migrations/001_init.sql**：创建所有表和索引

---

## 8. 如何启动项目

### 8.1 前置条件

- Docker Desktop 已安装并启动
- Go 1.21+ 已安装
- 终端（PowerShell / Git Bash / WSL）

### 8.2 启动步骤

```bash
# 第1步：进入项目目录
cd BlockExplore

# 第2步：复制配置文件
cp .env.example .env

# 第3步：编辑 .env 文件（可选，使用默认值也可以）

# 第4步：启动基础设施（PostgreSQL + Redis + Kafka）
docker compose up -d postgres redis kafka

# 第5步：等待容器健康（约 30 秒）
docker compose ps

# 第6步：编译 Go 服务
go build -o bin/ ./cmd/...

# 第7步：启动 API 服务
./bin/query-api.exe &        # 查询 API（端口 8080）
./bin/search-api.exe &       # 搜索 API（端口 8081）
./bin/price-api.exe &        # 价格 API（端口 8082）

# 第8步：启动数据处理服务
./bin/block-processor.exe &  # 区块处理器
./bin/eth-sync-worker.exe &  # 以太坊同步
```

### 8.3 验证服务

```bash
# 检查容器状态
docker compose ps

# 检查 Go 服务进程
tasklist | grep -i "query-api\|search-api\|price-api\|block-processor\|eth-sync"

# 测试 API
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/blocks?chain=eth
curl http://localhost:8080/api/v1/blocks/25178250?chain=eth
curl "http://localhost:8080/api/v1/blocks/25178250/transactions?chain=eth&page_size=5"
```

### 8.4 停止服务

```bash
# 停止 Go 服务
taskkill //F //IM query-api.exe
taskkill //F //IM search-api.exe
taskkill //F //IM price-api.exe
taskkill //F //IM block-processor.exe
taskkill //F //IM eth-sync-worker.exe

# 停止 Docker 容器
docker compose down

# 停止并删除数据（清空数据库）
docker compose down -v
```

---

## 9. API 接口说明

### 9.1 统一响应格式

所有 API 返回格式统一：

```json
{
  "code": 200,           // 状态码：200=成功，4xx=客户端错误，5xx=服务端错误
  "message": "success",  // 状态描述
  "data": {},            // 响应数据
  "request_id": "uuid"   // 请求 ID（用于日志追踪）
}
```

### 9.2 区块相关接口

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | /api/v1/blocks | 区块列表 | chain, page, page_size |
| GET | /api/v1/blocks/:number | 区块详情 | chain |
| GET | /api/v1/blocks/:number/transactions | 区块内交易 | chain, page, page_size |

**示例**：
```bash
# 获取以太坊最新 5 个区块
curl "http://localhost:8080/api/v1/blocks?chain=eth&page_size=5"

# 获取区块 25178250 的详情
curl "http://localhost:8080/api/v1/blocks/25178250?chain=eth"

# 获取区块 25178250 内的前 5 笔交易
curl "http://localhost:8080/api/v1/blocks/25178250/transactions?chain=eth&page_size=5"
```

### 9.3 交易相关接口

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | /api/v1/transactions/:hash | 交易详情 | chain |

### 9.4 地址相关接口

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | /api/v1/addresses/:address/transactions | 地址交易历史 | chain, page, page_size |

### 9.5 搜索接口

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | /api/v1/search | 统一搜索 | q (关键词) |

**示例**：
```bash
# 搜索区块号
curl "http://localhost:8080/api/v1/search?q=25178250"

# 搜索交易哈希
curl "http://localhost:8080/api/v1/search?q=0xf5ec7d35fd792b7ce81e51987faa4768b6eb12c6508ca8a4d4816fd3e12b7d28"
```

### 9.6 价格接口

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | /api/v1/price/:chain | 当前价格 | - |
| GET | /api/v1/price/:chain/history | 价格历史 | start_time, end_time, limit |

---

## 10. 常见问题

### Q1: 为什么区块数据同步这么慢？

**A**: 因为我们用的是公共 RPC 节点，有请求频率限制。如果需要更快，可以：
1. 使用付费 RPC 服务（如 Alchemy、Infura）
2. 搭建自己的全节点

### Q2: Kafka 是必须的吗？

**A**: 不是必须的，但用了有好处：
- 解耦同步和存储
- 数据不会因为数据库故障而丢失
- 可以扩展多个消费者并行处理

如果不想用 Kafka，可以让 Sync Worker 直接写数据库。

### Q3: Redis 是必须的吗？

**A**: 不是必须的，但用了能提升性能：
- 减少数据库查询次数
- 加快 API 响应速度

如果不用 Redis，Service 层会直接查数据库，功能不受影响。

### Q4: 如何添加新链的支持？

**A**: 以添加 Polygon (MATIC) 为例：
1. 在 internal/client/ 创建 polygon_client.go
2. 在 internal/service/sync/ 创建 polygon_sync.go
3. 在 cmd/ 创建 polygon-sync-worker/main.go
4. 在 .env 添加 Polygon 的 RPC 配置
5. 在 Kafka 创建新的 Topic

### Q5: Docker 容器启动失败怎么办？

**A**: 常见原因和解决方法：
1. 端口被占用：`netstat -ano | findstr :5432` 检查端口
2. 数据卷冲突：`docker compose down -v` 清除旧数据
3. 镜像拉取失败：检查网络或配置镜像加速器

### Q6: 如何查看日志？

**A**:
```bash
# Docker 容器日志
docker logs blockexplore-postgres
docker logs blockexplore-redis
docker logs blockexplore-kafka

# Go 服务日志（前台运行可以看到）
./bin/query-api.exe
```
