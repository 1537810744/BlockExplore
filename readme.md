# BlockExplore — 多链区块链浏览器

> 6.13 黑客松作品

一个支持 **Ethereum / Bitcoin / Solana** 三链的实时区块链浏览器，采用 Go 微服务 + Kafka 异步同步 + Next.js 前端，
从区块链节点拉取原始数据，经过解析、清洗、入库，最终以类似 Etherscan 的界面呈现。

## 技术栈

| 层 | 技术 | 用途 |
|---|---|---|
| 前端 | Next.js 14 · TypeScript · Tailwind CSS · Recharts | 页面渲染、价格图表 |
| 网关 | Next.js Rewrites | API 反向代理 |
| 后端 | Go · Gin · GORM | REST API、业务逻辑 |
| 消息队列 | Apache Kafka | 异步解耦、削峰填谷 |
| 缓存 | Redis | 热点数据缓存 |
| 数据库 | PostgreSQL | 区块、交易、价格持久化 |
| 容器化 | Docker · Docker Compose | 一键部署、环境隔离 |
| 外部 API | BlockCypher · Solana RPC · Ethereum RPC · CoinGecko | 区块链数据源 |

## 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         浏览器 (Next.js)                         │
│                Etherscan 风格深色主题 · 三链切换                 │
└──────────────┬──────────────────────────────────────────────────┘
               │  HTTP /api/v1/...
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        query-api (Go)                           │
│              Handler → Service → Repository → PostgreSQL        │
│                                  │                              │
│                                  ├── Redis 缓存                  │
│                                  └── 统一 JSON 响应              │
└─────────────────────────────────────────────────────────────────┘
               ▲                          ▲
               │                          │
┌──────────────┴──────────┐  ┌────────────┴──────────┐
│    search-api (Go)      │  │    price-api (Go)      │
│    全文搜索地址/交易哈希  │  │    价格查询 + CoinGecko │
└─────────────────────────┘  └─────────────────────────┘
               ▲                          ▲
               │                          │
               │    ┌─────────────────────┘
               │    │
┌──────────────┴────┴─────────────────────────────────────────────┐
│                      PostgreSQL · Redis                         │
└─────────────────────────────────────────────────────────────────┘
               ▲
               │  写入数据
               │
┌──────────────┴──────────────────────────────────────────────────┐
│                    block-processor (Go)                         │
│               Kafka Consumer · 解析 · 入库                      │
└─────────────────────────────────────────────────────────────────┘
               ▲
               │  Kafka 消息
               │
┌──────────────┴──────────────────────────────────────────────────┐
│                     Apache Kafka                                │
│          Topic: block.raw.eth / block.raw.btc / block.raw.sol   │
└─────────────────────────────────────────────────────────────────┘
               ▲          ▲          ▲
               │          │          │
┌──────────────┴─┐ ┌──────┴─┐ ┌──────┴──────────┐
│ eth-sync-worker│ │btc-sync│ │ sol-sync-worker  │
│    RPC 拉取     │ │  API   │ │   RPC 拉取       │
│   以太坊区块    │ │拉BTC区块│ │  Solana 区块     │
└──────┬─────────┘ └───┬────┘ └─────┬────────────┘
       │               │            │
       ▼               ▼            ▼
┌──────────────────────────────────────────────────┐
│              Ethereum · Bitcoin · Solana         │
│              公链节点 / BlockCypher API           │
└──────────────────────────────────────────────────┘
```

## 数据流：一条区块链数据是怎么到前端的

```
1. 拉取
   eth-sync-worker 调用 eth_getBlockByNumber
   → 拿到区块 + 交易的原始 JSON（十六进制 Wei 格式）

2. 发送
   封装为 BlockMessage → Kafka Producer
   → 写入 Topic "block.raw.eth"

3. 消费
   block-processor 从 Kafka 拉取 → 解析 JSON
   → Wei 转 ETH、Lamports 转 SOL、Satoshi 转 BTC
   → GORM INSERT 写入 PostgreSQL

4. 查询
   浏览器 GET /api/v1/blocks?chain=eth&page=1
   → Next.js Rewrites 代理到 query-api:8080
   → Handler 解析 HTTP 参数
   → Service 查 Redis 缓存（未命中查 PostgreSQL）
   → Repository 执行 SQL
   → JSON 返回: { blocks: [...], pagination: {...} }

5. 渲染
   Next.js 页面组件接收 JSON → React 渲染
   → 区块列表、交易详情、价格走势图
```

### 为什么要经过 Kafka？

```
不是直接写数据库：

  同步 Worker → PostgreSQL    ❌ 同步阻塞，慢查询拖慢拉取

而是：

  同步 Worker → Kafka → 入库 Worker → PostgreSQL   ✅ 解耦 + 削峰 + 异步
```

- 三个同步 Worker 可以并行拉取，不互相等待
- 入库 Worker 按自己的节奏消费，Kafka 当缓冲区
- 哪个环节挂了，重启继续消费，不丢数据

## 项目结构

```
BlockExplore/
├── cmd/                          # 7 个可执行程序入口
│   ├── query-api/                # 查询 API（HTTP 8080）
│   ├── search-api/               # 搜索 API（HTTP 8081）
│   ├── price-api/                # 价格 API（HTTP 8082）
│   ├── eth-sync-worker/          # ETH 区块同步
│   ├── btc-sync-worker/          # BTC 区块同步
│   ├── sol-sync-worker/          # SOL 区块同步
│   └── block-processor/          # Kafka 消费 + 入库
│
├── internal/                     # 内部包（三层架构）
│   ├── client/                   # 区块链 RPC 客户端
│   │   ├── eth_client.go         # 以太坊 JSON-RPC
│   │   ├── btc_client.go         # 比特币 BlockCypher API
│   │   └── sol_client.go         # Solana JSON-RPC
│   ├── handler/                  # HTTP 处理层
│   ├── service/                  # 业务逻辑层
│   │   ├── query/                # 查询服务
│   │   ├── sync/                 # 同步调度
│   │   ├── processor/            # 区块解析入库
│   │   └── price/                # 价格服务
│   ├── repository/               # 数据访问层（SQL）
│   ├── mq/                       # Kafka 生产者/消费者
│   ├── model/                    # 数据模型
│   ├── config/                   # 配置管理
│   └── router/                   # URL 路由
│
├── web/                          # Next.js 前端
│   ├── src/
│   │   ├── app/                  # App Router 页面
│   │   │   ├── page.tsx          # 首页（价格 + 最新区块）
│   │   │   ├── blocks/           # 区块列表 / 详情
│   │   │   ├── tx/               # 交易详情
│   │   │   └── address/          # 地址交易记录
│   │   ├── components/           # 组件
│   │   │   ├── Header.tsx        # 导航栏 + 搜索
│   │   │   ├── PriceChart.tsx    # 价格走势图
│   │   │   ├── BlockTable.tsx    # 区块列表
│   │   │   └── TxTable.tsx       # 交易列表
│   │   └── lib/
│   │       └── api.ts            # 类型安全 API 客户端
│   └── next.config.js            # Rewrites 代理配置
│
├── migrations/                   # 数据库初始化 SQL
├── docker-compose.yaml           # 生产环境（全部容器化）
├── docker-compose.dev.yaml       # 开发环境（仅基础设施）
├── Dockerfile                    # Go 后端镜像
├── .env                          # 本地开发配置
├── .env.docker                   # Docker 环境配置
├── ARCHITECTURE.md               # 架构详解
├── DOCKER.md                     # 容器化详解
└── INTERVIEW.md                  # 面试准备文档
```

## 三层架构

```
┌──────────┐       ┌──────────┐       ┌──────────────┐
│  Handler  │ ────→ │  Service  │ ────→ │  Repository  │
│          │ ←──── │          │ ←──── │              │
│ HTTP 参数 │       │ 业务逻辑  │       │  SQL / GORM   │
│ JSON 响应 │       │ 缓存策略  │       │  数据库交互    │
└──────────┘       └──────────┘       └──────────────┘
      知道：              知道：              知道：
  参数名、状态码      缓存、校验、规则       表名、索引、SQL
      不知道：            不知道：            不知道：
  缓存、SQL           HTTP 参数、SQL        HTTP 参数、业务
```

**核心价值：隔离变化。** 换数据库只改 Repository，换 HTTP 框架只改 Handler。

## 多链适配设计

三种链的差异很大，但对外暴露统一的 `Block` 和 `Transaction` 模型：

```go
// 所有链共用同一个模型
type Block struct {
    Chain       string   // "eth" / "btc" / "sol"
    BlockNumber int64
    BlockHash   string
    Timestamp   int64
    TxCount     int
    // ...
    Slot        *int64   // 只有 Solana 有（指针允许 nil）
}

type Transaction struct {
    Chain    string
    TxHash   string
    FromAddr string
    ToAddr   string
    Value    string   // 统一转为 ETH / BTC / SOL 单位
    // ...
}
```

| 差异 | ETH | BTC | SOL |
|---|---|---|---|
| 数据源 | JSON-RPC | BlockCypher REST | JSON-RPC |
| 金额单位 | Wei (10^18) | Satoshi (10^8) | Lamports (10^9) |
| 出块速度 | ~12 秒 | ~10 分钟 | ~0.4 秒 |
| Slot 概念 | 无 | 无 | 有 |
| 交易模型 | Account 模型 | UTXO 模型 | Account 模型 |

## 快速开始

### 开发环境

```bash
# 1. 启动基础设施（PostgreSQL + Redis + Kafka）
docker compose -f docker-compose.dev.yaml up -d

# 2. 等待健康检查通过
docker compose -f docker-compose.dev.yaml ps

# 3. 启动所有 Go 后端（开 7 个终端，或后台运行）
go run ./cmd/query-api/ &
go run ./cmd/search-api/ &
go run ./cmd/price-api/ &
go run ./cmd/eth-sync-worker/ &
go run ./cmd/btc-sync-worker/ &
go run ./cmd/sol-sync-worker/ &
go run ./cmd/block-processor/ &

# 4. 启动前端
cd web
npm install
npm run dev

# 5. 打开浏览器
# http://localhost:3000
```

### 生产部署

```bash
# 一键启动所有服务
docker compose up -d

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f
```

## 服务端口

| 服务 | 端口 | 说明 |
|---|---|---|
| Next.js 前端 | 3000 | Web 界面 |
| query-api | 8080 | 区块/交易/地址查询 |
| search-api | 8081 | 全文搜索 |
| price-api | 8082 | 价格查询 |
| PostgreSQL | 5432 | 数据库 |
| Redis | 6379 | 缓存 |
| Kafka | 9092 | 消息队列 |

## 11 个 Docker 服务

```
postgres          基础设施 - 数据库
redis             基础设施 - 缓存
kafka             基础设施 - 消息队列
query-api          查询 API（查区块/交易）
search-api         搜索 API（搜地址/TxHash）
price-api          价格 API（CoinGecko）
eth-sync-worker    ETH 区块同步
btc-sync-worker    BTC 区块同步
sol-sync-worker    SOL 区块同步
block-processor    区块入库处理
web               Next.js 前端
```

## 文档

- [架构详解](ARCHITECTURE.md) — 三层架构、分层原因、数据流
- [容器化详解](DOCKER.md) — Docker 部署、服务发现、配置机制
- [面试准备](INTERVIEW.md) — 涵盖全部技术栈的面试题
