# BlockExplore — 高并发多链区块链浏览器

## 项目概述

BlockExplore 是一个高并发区块链浏览器，支持 Ethereum（ETH）、Bitcoin（BTC）、Solana（SOL）三条链的区块与交易数据查询。系统从链上节点持续同步数据至本地数据库，并通过 RESTful API 对外提供高性能查询服务。

---

## 技术栈

| 层级 | 技术选型 | 说明 |
|------|---------|------|
| **语言** | Go 1.21+ | 高并发后端服务 |
| **HTTP 框架** | Gin | 高性能 RESTful API 路由 |
| **数据库** | PostgreSQL | 主存储，区块/交易/地址等结构化数据 |
| **缓存** | Redis | 热数据缓存、价格缓存、查询限流 |
| **消息队列** | Kafka | 链上数据采集与处理解耦 |
| **容器编排** | Docker Compose | 统一编排所有服务 |
| **反向代理** | Nginx | API 网关、负载均衡、限流 |
| **ORM** | GORM | 数据库访问层 |
| **配置管理** | Viper | 统一配置（.env / 环境变量） |
| **日志** | Zap | 高性能结构化日志 |
| **区块链 RPC 客户端** | go-ethereum / btcd / solana-go | 各链原生 SDK |
| **定时调度** | robfig/cron | 区块同步任务调度 |

### 基础设施组件

```
┌─────────────────────────────────────────────────────────┐
│                     Docker Compose                       │
├────────────┬────────────┬──────────┬──────────┬─────────┤
│   Nginx    │   Go API   │  Kafka   │  Redis   │   PG    │
│  (网关)     │  Services  │ (消息)    │ (缓存)    │ (存储)  │
│   :80      │  :8080~    │ :9092    │ :6379    │ :5432   │
└────────────┴────────────┴──────────┴──────────┴─────────┘
```

---

## 架构设计

### 整体架构

采用**微服务 + 异步管道**架构，核心思想：**先同步到本地库，再对外提供 API 查询**。

```
                          ┌──────────────┐
                          │    Nginx      │
                          │  (API 网关)   │
                          └──────┬───────┘
                                 │
                 ┌───────────────┼───────────────┐
                 │               │               │
          ┌──────▼──────┐ ┌─────▼─────┐ ┌──────▼──────┐
          │  Query API   │ │ Search API│ │  Price API  │
          │   Service    │ │  Service  │ │   Service   │
          │   :8080      │ │  :8081    │ │   :8082     │
          └──────┬───────┘ └─────┬─────┘ └──────┬──────┘
                 │               │               │
                 └───────────────┼───────────────┘
                                 │
                        ┌────────▼───────┐
                        │   PostgreSQL   │
                        │  (主存储)       │
                        └────────▲───────┘
                                 │
                 ┌───────────────┼───────────────┐
                 │               │               │
          ┌──────┴──────┐ ┌─────┴─────┐ ┌──────┴──────┐
          │  ETH Sync    │ │ BTC Sync  │ │  SOL Sync   │
          │   Worker     │ │  Worker   │ │   Worker    │
          └──────┬───────┘ └─────┬─────┘ └──────┬──────┘
                 │               │               │
          ┌──────▼──────┐ ┌─────▼─────┐ ┌──────▼──────┐
          │ ETH Full     │ │BTC Full   │ │ SOL Full    │
          │  Node (RPC)  │ │Node (RPC) │ │ Node (RPC)  │
          └──────────────┘ └───────────┘ └─────────────┘
```

### 微服务拆分

| 服务 | 职责 | 端口 |
|------|------|------|
| **eth-sync-worker** | 从以太坊全节点拉取区块/交易，写入 Kafka | — |
| **btc-sync-worker** | 从比特币全节点拉取区块/交易，写入 Kafka | — |
| **sol-sync-worker** | 从 Solana RPC 拉取区块/交易，写入 Kafka | — |
| **block-processor** | 消费 Kafka 消息，解析并持久化区块与交易数据到 PostgreSQL | — |
| **query-api** | 对外 RESTful API：区块列表、区块详情、交易列表 | 8080 |
| **search-api** | 统一搜索入口：支持地址/交易哈希/区块号/ENS 检索 | 8081 |
| **price-api** | 原生代币价格查询、历史价格曲线数据 | 8082 |

### 目录结构

```
BlockExplore
├── cmd/
│   ├── eth-sync-worker/main.go      # ETH 同步 Worker 入口
│   ├── btc-sync-worker/main.go      # BTC 同步 Worker 入口
│   ├── sol-sync-worker/main.go      # SOL 同步 Worker 入口
│   ├── block-processor/main.go      # 区块处理器入口
│   ├── query-api/main.go            # 查询 API 入口
│   ├── search-api/main.go           # 搜索 API 入口
│   └── price-api/main.go            # 价格 API 入口
├── internal/
│   ├── config/                      # 配置管理 (Viper)
│   │   └── config.go
│   ├── model/                       # 数据模型 (GORM)
│   │   ├── block.go
│   │   ├── transaction.go
│   │   └── address.go
│   ├── repository/                  # 数据访问层
│   │   ├── block_repo.go
│   │   ├── tx_repo.go
│   │   └── search_repo.go
│   ├── service/                     # 业务逻辑层
│   │   ├── sync/
│   │   │   ├── eth_sync.go
│   │   │   ├── btc_sync.go
│   │   │   └── sol_sync.go
│   │   ├── processor/
│   │   │   └── block_processor.go
│   │   ├── query/
│   │   │   └── query_service.go
│   │   └── price/
│   │       └── price_service.go
│   ├── handler/                     # HTTP Handler (Controller)
│   │   ├── block_handler.go
│   │   ├── tx_handler.go
│   │   ├── search_handler.go
│   │   └── price_handler.go
│   ├── router/                      # 路由注册
│   │   └── router.go
│   ├── middleware/                   # 中间件
│   │   ├── ratelimit.go
│   │   ├── cors.go
│   │   └── request_id.go
│   ├── client/                      # 区块链 RPC 客户端封装
│   │   ├── eth_client.go
│   │   ├── btc_client.go
│   │   └── sol_client.go
│   └── mq/                          # Kafka 生产者/消费者
│       ├── producer.go
│       └── consumer.go
├── pkg/                             # 公共工具库
│   ├── logger/                      # Zap 日志封装
│   ├── cache/                       # Redis 缓存工具
│   └── errcode/                     # 错误码定义
├── migrations/                      # 数据库迁移文件
├── docker-compose.yaml              # 服务编排
├── Dockerfile                       # Go 服务镜像
├── nginx.conf                       # Nginx 配置
├── .env.example                     # 配置模板
├── README.md                        # 项目说明
└── PROJECT_DOCUMENTATION.md         # 本文档
```

---

## 数据流

### 1. 区块同步数据流（异步管道）

```
Blockchain          Kafka              PostgreSQL          API
   Node              Topic              Tables            Service
    │                  │                   │                  │
    │ ① 拉取新区块      │                   │                  │
    ├─────────────────►│                   │                  │
    │  raw block data  │                   │                  │
    │                  │ ② 消费 & 解析     │                  │
    │                  ├─────────────────►│                  │
    │                  │  parsed block     │                  │
    │                  │  + transactions   │                  │
    │                  │                   │ ③ 索引 & 存储    │
    │                  │                   │                  │
    │                  │                   │     ④ 查询请求    │
    │                  │                   │◄─────────────────┤
    │                  │                   │     JSON 响应    │
    │                  │                   ├─────────────────►│
```

**详细步骤：**

1. **Sync Worker** 通过各链 RPC 接口每 ~12 秒（ETH 出块时间）/ ~10 分钟（BTC）/ ~0.4 秒（SOL）轮询或通过 WebSocket 订阅新区块
2. 拉取到新区块后，序列化为标准化消息，发送到 Kafka Topic：`block.raw.{chain}`
3. **Block Processor** 消费 Kafka 消息，解析原始区块数据，提取交易列表，写入 PostgreSQL
4. 同时将关键数据写入 Redis 缓存（最新 N 个区块、热门查询等）
5. **Query API** 查询时优先读 Redis 缓存，未命中则查 PostgreSQL

### 2. 搜索查询数据流

```
  Client                 Nginx              Search API          PostgreSQL
    │                      │                     │                   │
    │  GET /api/v1/search  │                     │                   │
    │  ?q={keyword}        │                     │                   │
    ├─────────────────────►│                     │                   │
    │                      │  route /api/v1/*    │                   │
    │                      ├────────────────────►│                   │
    │                      │                     │                   │
    │                      │                     │  ① 识别输入类型    │
    │                      │                     │  (address/tx/block)│
    │                      │                     │                   │
    │                      │                     │  ② 路由查询        │
    │                      │                     ├──────────────────►│
    │                      │                     │  query result      │
    │                      │                     │◄──────────────────┤
    │                      │                     │                   │
    │                      │  ③ 统一响应格式     │                   │
    │                      │◄────────────────────┤                   │
    │◄─────────────────────│                     │                   │
    │  200 OK              │                     │                   │
    │  { type, data }      │                     │                   │
```

**输入类型识别逻辑：**

| 输入模式 | 判定规则 | 查询目标 |
|---------|---------|---------|
| 42 字符，`0x` 开头 | 以太坊交易哈希 | `transactions` 表 |
| 64 字符，非 `0x` 开头 | 比特币交易哈希 | `transactions` 表 |
| 87-88 字符，Base58 | Solana 签名 | `transactions` 表 |
| `0x` + 40 字符 | 以太坊地址 | `addresses` 表 |
| `1`/`3`/`bc1` 开头 | 比特币地址 | `addresses` 表 |
| 纯数字 | 区块高度 | `blocks` 表 |

### 3. 价格曲线数据流

```
  Price API        CoinGecko / CoinMarketCap        Redis           Client
    │                       │                         │               │
    │ ① 定时拉取 (30s)      │                         │               │
    ├──────────────────────►│                         │               │
    │  价格数据 JSON         │                         │               │
    │◄──────────────────────┤                         │               │
    │                       │                         │               │
    │ ② 写入缓存 (TTL 60s)  │                         │               │
    ├───────────────────────────────────────────────►│               │
    │                       │                         │               │
    │                       │        ③ GET /price     │               │
    │                       │◄────────────────────────┤               │
    │                       │        价格数据          │               │
    │                       ├────────────────────────►│               │
```

实时价格每 30 秒从外部 API 拉取一次，历史价格曲线数据（K 线）定时写入 PostgreSQL 的 `price_history` 表。

---

## 数据库设计

### 核心表结构

```sql
-- 区块表
CREATE TABLE blocks (
    id              BIGSERIAL PRIMARY KEY,
    chain           VARCHAR(10) NOT NULL,      -- 'eth' | 'btc' | 'sol'
    block_number    BIGINT NOT NULL,
    block_hash      VARCHAR(128) NOT NULL,
    parent_hash     VARCHAR(128),
    timestamp       BIGINT NOT NULL,            -- Unix timestamp
    tx_count        INT DEFAULT 0,
    gas_used        NUMERIC(40,0),             -- ETH/SOL
    gas_limit       NUMERIC(40,0),             -- ETH
    size_bytes      INT,                        -- BTC
    difficulty      NUMERIC(40,0),              -- BTC
    slot            BIGINT,                     -- SOL
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(chain, block_number),
    UNIQUE(chain, block_hash)
);

CREATE INDEX idx_blocks_chain_number ON blocks(chain, block_number DESC);
CREATE INDEX idx_blocks_timestamp ON blocks(chain, timestamp DESC);

-- 交易表
CREATE TABLE transactions (
    id              BIGSERIAL PRIMARY KEY,
    chain           VARCHAR(10) NOT NULL,
    tx_hash         VARCHAR(128) NOT NULL,
    block_number    BIGINT NOT NULL,
    block_id        BIGINT REFERENCES blocks(id),
    from_addr       VARCHAR(128),
    to_addr         VARCHAR(128),
    value           NUMERIC(78,18),
    gas_price       NUMERIC(40,0),
    gas_used        NUMERIC(40,0),
    gas_limit       NUMERIC(40,0),
    nonce           BIGINT,
    input_data      TEXT,                       -- ETH calldata
    status          SMALLINT DEFAULT 1,         -- 1=success 0=fail
    timestamp       BIGINT NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(chain, tx_hash)
);

CREATE INDEX idx_tx_block ON transactions(block_id);
CREATE INDEX idx_tx_from ON transactions(from_addr);
CREATE INDEX idx_tx_to ON transactions(to_addr);
CREATE INDEX idx_tx_hash ON transactions(chain, tx_hash);

-- 价格历史表
CREATE TABLE price_history (
    id              BIGSERIAL PRIMARY KEY,
    chain           VARCHAR(10) NOT NULL,
    symbol          VARCHAR(10) NOT NULL,
    price_usd       NUMERIC(20,8),
    timestamp       BIGINT NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_price_chain_time ON price_history(chain, timestamp DESC);
```

---

## API 设计

### 统一响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "request_id": "uuid-v4"
}
```

### 接口列表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/blocks` | 区块列表（分页） |
| GET | `/api/v1/blocks/{block_number}` | 区块详情 |
| GET | `/api/v1/blocks/{block_number}/transactions` | 区块内交易列表 |
| GET | `/api/v1/transactions/{hash}` | 交易详情 |
| GET | `/api/v1/addresses/{address}` | 地址概览 |
| GET | `/api/v1/addresses/{address}/transactions` | 地址交易历史 |
| GET | `/api/v1/search` | 统一搜索 |
| GET | `/api/v1/price/{chain}` | 当前价格 |
| GET | `/api/v1/price/{chain}/history` | 历史价格曲线 |

### 区块列表接口（核心）

```
GET /api/v1/blocks?chain=eth&page=1&page_size=20
```

响应：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "chain": "eth",
    "blocks": [
      {
        "block_number": 18500421,
        "block_hash": "0xabc...",
        "timestamp": 1700000000,
        "tx_count": 156,
        "gas_used": "12345678",
        "gas_limit": "30000000",
        "validator": "0xdef..."
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 18500421
    }
  },
  "request_id": "uuid-v4"
}
```

---

## 部署架构

### Docker Compose 编排

```
                   ┌──────────────────────────────────────┐
                   │            Nginx :80                  │
                   │      (反向代理 + 限流 + 缓存)          │
                   └────┬─────┬─────┬─────┬───────────────┘
                        │     │     │     │
        ┌───────────────┤     │     │     ├───────────────┐
        │               │     │     │     │               │
        ▼               ▼     ▼     ▼     ▼               ▼
  ┌──────────┐   ┌──────────┐ ┌──────────────┐  ┌──────────────┐
  │query-api │   │search-api│ │ block-proc   │  │  price-api   │
  │  :8080   │   │  :8081   │ │   :8082      │  │   :8083      │
  └────┬─────┘   └────┬─────┘ └──────┬───────┘  └──────┬───────┘
       │              │              │                  │
       └──────────────┼──────────────┼──────────────────┘
                      │              │
              ┌───────▼──────┐ ┌────▼────────┐
              │  PostgreSQL  │ │    Redis     │
              │    :5432     │ │    :6379     │
              └──────────────┘ └─────────────┘
                      ▲
                      │
              ┌───────┴──────────────┐
              │       Kafka          │
              │      :9092           │
              └───────▲──────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
  ┌─────┴──────┐ ┌────┴─────┐ ┌────┴──────┐
  │eth-sync    │ │btc-sync  │ │sol-sync   │
  │  worker    │ │ worker   │ │  worker   │
  └────────────┘ └──────────┘ └───────────┘
```

---

## 高并发设计要点

| 策略 | 实现方式 |
|------|---------|
| **读写分离** | Sync Worker 只写，API 只读，互不阻塞 |
| **多级缓存** | Nginx 缓存 → Redis 热点缓存 → PostgreSQL |
| **异步解耦** | Kafka 消息队列解耦同步与处理 |
| **连接池** | GORM 连接池 + Redis 连接池 |
| **分页查询** | 游标分页避免大 OFFSET，默认 page_size ≤ 50 |
| **限流** | Nginx `limit_req_zone` + API 层令牌桶 |
| **索引优化** | 覆盖查询的联合索引，避免全表扫描 |
| **批量写入** | Block Processor 批量 INSERT 区块和交易 |

---

## 参考规范

本项目参考以下开源项目的工程规范：

- [Epusdt](https://github.com/GMWalletApp/epusdt) — Go 微服务架构、配置管理、Docker 部署模式
- [Etherscan](https://etherscan.io/) — 区块链浏览器 API 设计参考
- [Blockchair](https://blockchair.com/) — 多链浏览器数据展示参考
