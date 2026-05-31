# BlockExplore 项目完全指南

> 这份文档是写给"看不懂代码的人"的。它会告诉你：这个项目是什么、怎么设计的、数据怎么流动、程序从哪里启动到哪里结束。看完这份文档再去看代码，就会豁然开朗。

---

## 目录

1. [项目是什么](#1-项目是什么)
2. [整体架构图](#2-整体架构图)
3. [文件夹结构详解](#3-文件夹结构详解)
4. [数据流：从区块链到你的屏幕](#4-数据流从区块链到你的屏幕)
5. [9 个程序的启动路径](#5-9-个程序的启动路径)
6. [核心模块逐个拆解](#6-核心模块逐个拆解)
7. [数据库表结构](#7-数据库表结构)
8. [前端结构](#8-前端结构)
9. [Docker 部署架构](#9-docker-部署架构)
10. [如何阅读代码](#10-如何阅读代码)

---

## 1. 项目是什么

BlockExplore 是一个**多链区块链浏览器**，类似于 etherscan.io，但支持三条链：

| 链 | 代币 | 出块时间 | RPC 节点 |
|---|---|---|---|
| Ethereum (ETH) | ETH | ~12 秒 | publicnode.com |
| Bitcoin (BTC) | BTC | ~10 分钟 | 需要本地节点 |
| Solana (SOL) | SOL | ~0.4 秒 | api.mainnet-beta.solana.com |

**用户能做什么：**
- 看到最新区块列表（从新到旧）
- 点击区块，看到区块里的所有交易
- 搜索区块号、交易哈希、地址
- 看到 ETH/BTC/SOL 的实时价格和价格曲线
- 切换链（BTC/ETH/SOL 面板）

---

## 2. 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户浏览器                                │
│                   http://localhost:3000                          │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                   nginx (前端容器)                                │
│           提供静态文件 + 反向代理 API 请求                         │
│                                                                  │
│   /                → 返回 index.html (React SPA)                │
│   /api/v1/blocks   → proxy_pass http://query-api:8080           │
│   /api/v1/search   → proxy_pass http://search-api:8081          │
│   /api/v1/price    → proxy_pass http://price-api:8082           │
└────┬─────────────────────┬──────────────────────┬───────────────┘
     │                     │                      │
     ▼                     ▼                      ▼
┌──────────┐      ┌──────────────┐      ┌──────────────┐
│query-api │      │ search-api   │      │  price-api   │
│  :8080   │      │   :8081      │      │   :8082      │
│ 区块/交易 │      │   统一搜索    │      │  价格查询     │
└────┬─────┘      └──────┬───────┘      └──────┬───────┘
     │                   │                      │
     ▼                   ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     PostgreSQL (5432)                            │
│              存储：blocks, transactions, addresses,               │
│                    price_history                                  │
└─────────────────────────────────────────────────────────────────┘
     ▲
     │ 写入数据
     │
┌────┴────────────────────────────────────────────────────────────┐
│                  block-processor                                 │
│            从 Kafka 消费消息，写入数据库                            │
└────┬────────────────────────────────────────────────────────────┘
     │
     │ 消费消息
     ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Kafka (9092)                                 │
│                                                                  │
│   Topic: block.raw.eth   (以太坊区块数据)                        │
│   Topic: block.raw.btc   (比特币区块数据)                        │
│   Topic: block.raw.sol   (Solana 区块数据)                       │
└────▲──────────────────────▲──────────────────────▲──────────────┘
     │                      │                      │
     │ 生产消息              │ 生产消息              │ 生产消息
     │                      │                      │
┌────┴─────┐       ┌───────┴──────┐       ┌───────┴──────┐
│eth-sync  │       │  btc-sync    │       │  sol-sync    │
│ worker   │       │   worker     │       │   worker     │
└────┬─────┘       └───────┬──────┘       └───────┬──────┘
     │                     │                      │
     │ JSON-RPC            │ JSON-RPC             │ JSON-RPC
     ▼                     ▼                      ▼
  以太坊节点            比特币节点              Solana 节点
(publicnode.com)      (需要本地运行)          (solana.com)
```

**一句话总结数据流：**
```
区块链节点 → sync-worker → Kafka → block-processor → PostgreSQL → query-api → nginx → 浏览器
```

---

## 3. 文件夹结构详解

```
BlockExplore/
├── cmd/                          ← 7 个微服务的入口（main.go）
│   ├── query-api/main.go         ← 查询 API 入口，端口 8080
│   ├── search-api/main.go        ← 搜索 API 入口，端口 8081
│   ├── price-api/main.go         ← 价格 API 入口，端口 8082
│   ├── eth-sync-worker/main.go   ← 以太坊同步 Worker 入口
│   ├── btc-sync-worker/main.go   ← 比特币同步 Worker 入口
│   ├── sol-sync-worker/main.go   ← Solana 同步 Worker 入口
│   └── block-processor/main.go   ← 区块处理器入口
│
├── internal/                     ← 所有业务代码（Go 约定：internal 外部不可引用）
│   ├── config/config.go          ← 配置管理（读取 .env 文件）
│   ├── model/                    ← 数据模型（对应数据库表）
│   │   ├── block.go              ← Block 结构体 → blocks 表
│   │   ├── transaction.go        ← Transaction 结构体 → transactions 表
│   │   ├── address.go            ← Address 结构体 → addresses 表
│   │   └── price.go              ← PriceHistory 结构体 → price_history 表
│   ├── client/                   ← 区块链 RPC 客户端
│   │   ├── eth_client.go         ← 以太坊 JSON-RPC 客户端
│   │   ├── btc_client.go         ← 比特币 JSON-RPC 客户端
│   │   └── sol_client.go         ← Solana JSON-RPC 客户端
│   ├── mq/                       ← Kafka 消息队列封装
│   │   ├── producer.go           ← 生产者（发送消息到 Kafka）
│   │   └── consumer.go           ← 消费者（从 Kafka 读取消息）
│   ├── repository/               ← 数据访问层（DAO，操作数据库）
│   │   ├── block_repo.go         ← 区块 CRUD 操作
│   │   ├── tx_repo.go            ← 交易 CRUD 操作
│   │   ├── search_repo.go        ← 统一搜索逻辑
│   │   └── price_repo.go         ← 价格数据操作
│   ├── service/                  ← 业务逻辑层
│   │   ├── sync/                 ← 同步服务（从区块链拉数据）
│   │   │   ├── eth_sync.go       ← 以太坊同步逻辑
│   │   │   ├── btc_sync.go       ← 比特币同步逻辑
│   │   │   └── sol_sync.go       ← Solana 同步逻辑
│   │   ├── processor/
│   │   │   └── block_processor.go← 从 Kafka 消费，写入数据库
│   │   ├── query/
│   │   │   └── query_service.go  ← 查询服务（带 Redis 缓存）
│   │   └── price/
│   │       └── price_service.go  ← 价格服务（CoinGecko API）
│   ├── handler/                  ← HTTP 处理器（Controller 层）
│   │   ├── block_handler.go      ← 区块相关接口
│   │   ├── tx_handler.go         ← 交易相关接口
│   │   ├── search_handler.go     ← 搜索接口
│   │   └── price_handler.go      ← 价格接口
│   ├── router/router.go          ← 路由注册（URL → Handler 映射）
│   └── middleware/               ← 中间件
│       ├── cors.go               ← 跨域支持
│       ├── request_id.go         ← 请求 ID（UUID）
│       └── ratelimit.go          ← 限流（令牌桶算法）
│
├── pkg/                          ← 通用工具包（可被外部引用）
│   ├── cache/redis.go            ← Redis 客户端封装
│   ├── logger/logger.go          ← Zap 日志封装
│   └── errcode/errcode.go        ← 错误码和统一响应格式
│
├── web/                          ← 前端项目（React + TypeScript）
│   ├── src/
│   │   ├── main.tsx              ← 前端入口
│   │   ├── App.tsx               ← 路由定义
│   │   ├── api/client.ts         ← API 调用封装
│   │   ├── types/index.ts        ← TypeScript 类型定义
│   │   ├── context/ChainContext.tsx ← 链切换上下文
│   │   ├── components/           ← 公共组件
│   │   │   ├── Layout.tsx        ← 页面布局
│   │   │   ├── ChainSwitcher.tsx ← 链切换按钮
│   │   │   ├── SearchBar.tsx     ← 搜索栏
│   │   │   ├── PriceCard.tsx     ← 价格卡片
│   │   │   └── PriceChart.tsx    ← 价格曲线图
│   │   └── pages/                ← 页面组件
│   │       ├── BlockList.tsx     ← 首页（区块列表 + 价格曲线）
│   │       ├── BlockDetail.tsx   ← 区块详情
│   │       ├── TxDetail.tsx      ← 交易详情
│   │       └── AddressTx.tsx     ← 地址交易历史
│   ├── Dockerfile                ← 前端 Docker 镜像
│   └── nginx.conf                ← 前端 nginx 配置
│
├── migrations/
│   └── 001_init.sql              ← 数据库建表 SQL
│
├── .env                          ← 本地开发配置（localhost）
├── .env.docker                   ← Docker 部署配置（服务名）
├── docker-compose.yaml           ← Docker 编排（9 个容器）
├── Dockerfile                    ← Go 服务的 Docker 镜像
└── nginx.conf                    ← 反向代理配置（未使用，用前端 nginx 代替）
```

---

## 4. 数据流：从区块链到你的屏幕

### 4.1 写入路径（数据怎么进数据库）

```
第 1 步：eth-sync-worker 启动
         │
         │  每 12 秒执行一次
         ▼
第 2 步：调用以太坊 RPC 接口
         │  POST https://ethereum.publicnode.com
         │  方法: eth_getBlockByNumber
         │  参数: 最新区块高度（十六进制）
         ▼
第 3 步：以太坊节点返回 JSON
         │  {
         │    "number": "0x18a4b3c",
         │    "hash": "0xabc...",
         │    "transactions": [...]
         │  }
         ▼
第 4 步：EthClient 解析 JSON
         │  十六进制 "0x18a4b3c" → 十进制 25828156
         │  构建 model.Block 和 []model.Transaction
         ▼
第 5 步：封装成 BlockMessage，发送到 Kafka
         │  Topic: block.raw.eth
         │  Key: "eth-25828156"
         │  Value: {"chain":"eth","block_number":25828156,"data":{...}}
         ▼
第 6 步：block-processor 从 Kafka 消费消息
         │  消费者组: block-processor-group
         │  三个 goroutine 并发消费三个 Topic
         ▼
第 7 步：BlockProcessor.Handle() 处理消息
         │  1. 反序列化 JSON → model.Block
         │  2. 反序列化 JSON → []model.Transaction
         │  3. 保存区块到 PostgreSQL (blocks 表)
         │  4. 设置交易的 BlockID 外键
         │  5. 批量保存交易到 PostgreSQL (transactions 表)
         ▼
第 8 步：数据已在 PostgreSQL 中
         SELECT * FROM blocks WHERE chain='eth' ORDER BY block_number DESC;
```

### 4.2 读取路径（数据怎么到前端）

```
第 1 步：用户打开浏览器 http://localhost:3000
         │
         ▼
第 2 步：nginx 返回 index.html（React SPA 单页应用）
         │  浏览器加载 JS/CSS
         ▼
第 3 步：React 渲染 BlockList 组件
         │  调用 api/client.ts 的 getBlockList('eth', 1, 20)
         ▼
第 4 步：axios 发送 HTTP 请求
         │  GET /api/v1/blocks?chain=eth&page=1&page_size=20
         ▼
第 5 步：nginx 反向代理
         │  /api/v1/blocks → http://query-api:8080
         ▼
第 6 步：query-api 的 Gin 路由匹配
         │  GET /api/v1/blocks → blockHandler.GetBlockList
         ▼
第 7 步：BlockHandler.GetBlockList()
         │  1. 解析参数: chain=eth, page=1, pageSize=20
         │  2. 调用 queryService.GetBlockList("eth", 1, 20)
         ▼
第 8 步：QueryService.GetBlockList()（带缓存）
         │  1. 先查 Redis: GET blocks:eth:1:20
         │     ├── 命中 → 直接返回缓存数据
         │     └── 未命中 → 继续
         │  2. 查 PostgreSQL: blockRepo.GetList("eth", 1, 20)
         │  3. 写入 Redis: SET blocks:eth:1:20 (过期时间 30 秒)
         │  4. 返回数据
         ▼
第 9 步：BlockHandler 构建 JSON 响应
         │  {
         │    "code": 200,
         │    "message": "success",
         │    "data": {
         │      "chain": "eth",
         │      "blocks": [...],
         │      "pagination": {"page":1, "page_size":20, "total":275}
         │    }
         │  }
         ▼
第 10 步：React 收到 JSON，渲染区块列表表格
          用户看到区块高度、时间、交易数、矿工、Gas 使用量
```

### 4.3 价格数据流

```
price-api 启动
    │
    │  每 30 秒执行一次
    ▼
调用 CoinGecko API
    │  GET https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd
    │  返回: {"ethereum": {"usd": 2030.31}}
    ▼
保存到 PostgreSQL (price_history 表)
    │
    ▼
更新 Redis 缓存
    │  SET price:current:eth (过期时间 60 秒)
    ▼
前端请求价格
    │  GET /api/v1/price/eth
    ▼
price-api 返回当前价格
    │  GET /api/v1/price/eth/history
    ▼
price-api 返回历史价格列表
    ▼
React 使用 recharts 绘制价格曲线
```

---

## 5. 9 个程序的启动路径

每个 `cmd/*/main.go` 都是一个独立程序，有自己的 `main()` 函数。下面逐个说明。

### 5.1 query-api（查询 API，端口 8080）

**职责：** 提供区块、交易的 RESTful 查询接口

**启动路径：**
```
cmd/query-api/main.go:main()
    │
    ├── 1. config.Load()              ← 读取 .env 配置
    ├── 2. logger.Init()              ← 初始化 Zap 日志
    ├── 3. gorm.Open(postgres)        ← 连接 PostgreSQL
    ├── 4. cache.Init()               ← 连接 Redis
    ├── 5. 创建各层实例:
    │       blockRepo := repository.NewBlockRepo(db)
    │       txRepo := repository.NewTxRepo(db)
    │       queryService := query.NewQueryService(blockRepo, txRepo, redisClient)
    │       blockHandler := handler.NewBlockHandler(queryService)
    │       txHandler := handler.NewTxHandler(queryService)
    ├── 6. router.Setup()             ← 注册路由
    │       GET /api/v1/blocks         → blockHandler.GetBlockList
    │       GET /api/v1/blocks/:number → blockHandler.GetBlockDetail
    │       GET /api/v1/transactions/:hash → txHandler.GetTransactionDetail
    └── 7. r.Run(":8080")             ← 启动 HTTP 服务（阻塞）
```

**终点：** 程序一直运行，直到被 kill 或收到 SIGTERM 信号。

### 5.2 search-api（搜索 API，端口 8081）

**职责：** 统一搜索（自动识别区块号/交易哈希/地址）

**启动路径：**
```
cmd/search-api/main.go:main()
    │
    ├── 1-3. 同 query-api（配置、日志、数据库）
    ├── 4. searchRepo := repository.NewSearchRepo(db)
    ├── 5. searchHandler := handler.NewSearchHandler(searchRepo)
    ├── 6. 注册路由: GET /api/v1/search?q=keyword → searchHandler.Search
    └── 7. r.Run(":8081")
```

**搜索逻辑（search_repo.go）：**
```
输入: "25212557"  → 纯数字 → 按区块号查询 blocks 表
输入: "0xabc..."  → 66 字符 → 按交易哈希查询 transactions 表
输入: "0xabc..."  → 42 字符 → 按地址查询 addresses 表
```

### 5.3 price-api（价格 API，端口 8082）

**职责：** 查询代币价格和价格历史，定时从 CoinGecko 同步

**启动路径：**
```
cmd/price-api/main.go:main()
    │
    ├── 1-3. 配置、日志、数据库
    ├── 4. Redis 缓存
    ├── 5. priceRepo + priceService
    ├── 6. 注册路由:
    │       GET /api/v1/price/:chain         → 价格查询
    │       GET /api/v1/price/:chain/history → 价格历史
    ├── 7. 启动定时任务（cron）
    │       每 30 秒调用 priceService.SyncPrices()
    └── 8. r.Run(":8082")
```

### 5.4 eth-sync-worker（以太坊同步）

**职责：** 从以太坊节点拉取区块数据，发送到 Kafka

**启动路径：**
```
cmd/eth-sync-worker/main.go:main()
    │
    ├── 1. config.Load()
    ├── 2. logger.Init()
    ├── 3. ethClient := client.NewEthClient("https://ethereum.publicnode.com")
    ├── 4. producer := mq.NewETHProducer(kafkaConfig)
    │       → 创建 Kafka Writer，Topic: block.raw.eth
    ├── 5. worker := sync.NewEthSyncWorker(ethClient, producer, 12)
    ├── 6. ctx, cancel := context.WithCancel()
    ├── 7. signal.Notify(sigChan, SIGINT, SIGTERM)  ← 监听 Ctrl+C
    └── 8. worker.Run(ctx)  ← 阻塞运行
            │
            ├── 立即执行一次 sync()
            └── 每 12 秒执行一次 sync()
                    │
                    ├── ethClient.GetLatestBlockNumber()
                    │       POST → eth_blockNumber → 返回最新高度
                    ├── ethClient.GetBlockByNumber(height)
                    │       POST → eth_getBlockByNumber → 返回区块+交易
                    └── producer.Send(ctx, blockMessage)
                            → 发送到 Kafka Topic: block.raw.eth
```

**终点：** 用户按 Ctrl+C → 收到 SIGINT → cancel() → worker.Run 返回 → 程序退出

### 5.5 block-processor（区块处理器）

**职责：** 从 Kafka 消费消息，解析后写入 PostgreSQL

**启动路径：**
```
cmd/block-processor/main.go:main()
    │
    ├── 1-3. 配置、日志、数据库
    ├── 4. blockRepo + txRepo + blockProcessor
    ├── 5. 创建 3 个 Kafka 消费者:
    │       consumer1 → Topic: block.raw.eth
    │       consumer2 → Topic: block.raw.btc
    │       consumer3 → Topic: block.raw.sol
    ├── 6. ctx, cancel := context.WithCancel()
    ├── 7. signal.Notify(sigChan, SIGINT, SIGTERM)
    └── 8. mq.ConsumeAll(ctx, consumers, blockProcessor.Handle)
            │
            ├── goroutine 1: consumer1.Consume(ctx, handler)
            │       循环: ReadMessage → Unmarshal → handler(msg)
            ├── goroutine 2: consumer2.Consume(ctx, handler)
            └── goroutine 3: consumer3.Consume(ctx, handler)
                    │
                    ▼
                blockProcessor.Handle(msg)
                    │
                    ├── json.Unmarshal(msg.Data) → Block + Transactions
                    ├── blockRepo.CreateSingle(&block)  ← 写入 blocks 表
                    ├── 设置交易的 BlockID 外键
                    └── txRepo.Create(transactions)     ← 写入 transactions 表
```

**终点：** Ctrl+C → cancel() → 所有消费者停止 → ConsumeAll 返回 → 程序退出

### 5.6 btc-sync-worker / sol-sync-worker

与 eth-sync-worker 结构完全相同，只是：
- 使用不同的 RPC 客户端（BtcClient / SolClient）
- 发送到不同的 Kafka Topic（block.raw.btc / block.raw.sol）
- 同步间隔不同（BTC: 600 秒，SOL: 1 秒）

---

## 6. 核心模块逐个拆解

### 6.1 分层架构（三层）

```
┌─────────────────────────────────────────────┐
│  Handler 层（handler/*.go）                  │
│  职责：解析 HTTP 参数，调用 Service，返回 JSON │
│  类比：Java 的 Controller                    │
├─────────────────────────────────────────────┤
│  Service 层（service/**/*.go）               │
│  职责：业务逻辑，缓存策略，外部 API 调用       │
│  类比：Java 的 Service                       │
├─────────────────────────────────────────────┤
│  Repository 层（repository/*.go）            │
│  职责：数据库 CRUD 操作                       │
│  类比：Java 的 DAO / Repository              │
└─────────────────────────────────────────────┘
```

**调用链示例（获取区块列表）：**
```
BlockHandler.GetBlockList(c *gin.Context)     ← handler 层
    │
    ├── 解析参数: chain, page, pageSize
    ├── 调用 queryService.GetBlockList(chain, page, pageSize)  ← service 层
    │       │
    │       ├── 查 Redis: GET blocks:eth:1:20
    │       ├── 未命中 → 调用 blockRepo.GetList(chain, page, pageSize)  ← repository 层
    │       │       │
    │       │       └── SELECT * FROM blocks WHERE chain='eth'
    │       │           ORDER BY block_number DESC LIMIT 20 OFFSET 0
    │       │
    │       └── 写 Redis: SET blocks:eth:1:20 (30s 过期)
    │
    └── c.JSON(200, errcode.Success(result))
```

### 6.2 Gin 路由框架

Gin 是 Go 语言最流行的 Web 框架，类似于 Python 的 Flask、Java 的 Spring MVC。

```go
// router/router.go 中的路由注册
r := gin.New()                    // 创建 Gin 引擎
r.Use(middleware.CORS())          // 注册全局中间件

v1 := r.Group("/api/v1")          // 创建路由组
{
    blocks := v1.Group("/blocks")  // 子路由组
    {
        blocks.GET("", blockHandler.GetBlockList)                    // GET /api/v1/blocks
        blocks.GET("/:block_number", blockHandler.GetBlockDetail)    // GET /api/v1/blocks/123
    }
}
```

**请求处理流程：**
```
HTTP 请求 → 中间件链 → 路由匹配 → Handler 函数 → JSON 响应
             │
             ├── RequestID 中间件：生成 UUID 请求 ID
             ├── CORS 中间件：设置跨域头
             ├── Recovery 中间件：捕获 panic
             └── RateLimit 中间件：令牌桶限流
```

### 6.3 Redis 缓存策略（Cache-Aside 模式）

```
读数据：
    1. 先查 Redis → 命中则直接返回（快！）
    2. Redis 未命中 → 查 PostgreSQL
    3. 查到后写入 Redis（设置过期时间）
    4. 返回数据

写数据：
    1. 写入 PostgreSQL
    2. 删除 Redis 缓存（下次读取时会重新加载）
```

**缓存键示例：**
```
blocks:eth:1:20        → 以太坊第 1 页每页 20 条的区块列表（30 秒过期）
block:eth:25212557     → 以太坊区块 25212557 的详情（60 秒过期）
tx:eth:0xabc...        → 以太坊交易详情（60 秒过期）
price:current:eth      → 以太坊当前价格（60 秒过期）
```

### 6.4 Kafka 消息队列

```
生产者（sync-worker）                    消费者（block-processor）
    │                                        │
    │  Send(BlockMessage)                    │  ReadMessage()
    │  ─────────────────────────────────►    │
    │                                        │
    │  Topic: block.raw.eth                  │  消费者组: block-processor-group
    │  Key: "eth-25828156"                   │  每条消息只被组内一个消费者处理
    │  Value: {"chain":"eth",...}            │
```

**为什么用 Kafka？**
- **解耦：** sync-worker 和 block-processor 互不依赖
- **削峰：** 区块数据量大时，Kafka 缓冲消息，防止数据库压力过大
- **可靠：** 消息持久化，即使消费者挂了也不会丢失

### 6.5 区块链 RPC 客户端

以太坊使用 JSON-RPC 2.0 协议通信：

```
请求：
POST https://ethereum.publicnode.com
Content-Type: application/json

{
    "jsonrpc": "2.0",
    "method": "eth_getBlockByNumber",
    "params": ["0x18a4b3c", true],
    "id": 1
}

响应：
{
    "jsonrpc": "2.0",
    "result": {
        "number": "0x18a4b3c",
        "hash": "0xabc...",
        "timestamp": "0x665f1234",
        "transactions": [...]
    },
    "id": 1
}
```

**关键转换：** 以太坊返回的数值都是十六进制（如 "0x18a4b3c"），需要转换为十进制（25828156）。

---

## 7. 数据库表结构

### 7.1 blocks 表（区块表）

```sql
CREATE TABLE blocks (
    id           BIGSERIAL PRIMARY KEY,   -- 自增主键
    chain        VARCHAR(10) NOT NULL,    -- 链标识: eth/btc/sol
    block_number BIGINT NOT NULL,         -- 区块高度
    block_hash   VARCHAR(128) NOT NULL,   -- 区块哈希
    parent_hash  VARCHAR(128),            -- 父区块哈希
    timestamp    BIGINT NOT NULL,         -- 出块时间（Unix 时间戳）
    tx_count     INT DEFAULT 0,           -- 交易数量
    gas_used     TEXT,                    -- Gas 使用量（ETH）
    gas_limit    TEXT,                    -- Gas 上限（ETH）
    size_bytes   INT,                     -- 区块大小（BTC）
    difficulty   TEXT,                    -- 难度（BTC）
    slot         BIGINT,                  -- 槽位号（SOL）
    created_at   TIMESTAMP DEFAULT NOW(), -- 记录创建时间
    UNIQUE(chain, block_number),          -- 同链区块高度唯一
    UNIQUE(chain, block_hash)             -- 同链区块哈希唯一
);
```

### 7.2 transactions 表（交易表）

```sql
CREATE TABLE transactions (
    id           BIGSERIAL PRIMARY KEY,
    chain        VARCHAR(10) NOT NULL,
    tx_hash      VARCHAR(128) NOT NULL,   -- 交易哈希
    block_number BIGINT NOT NULL,
    block_id     BIGINT REFERENCES blocks(id),  -- 外键关联区块
    from_addr    VARCHAR(128),            -- 发送方地址
    to_addr      VARCHAR(128),            -- 接收方地址
    value        TEXT,                    -- 转账金额
    gas_price    TEXT,                    -- Gas 价格
    gas_used     TEXT,                    -- 实际 Gas 消耗
    status       SMALLINT DEFAULT 1,      -- 1=成功 0=失败
    timestamp    BIGINT NOT NULL,
    UNIQUE(chain, tx_hash)
);
```

### 7.3 price_history 表（价格历史表）

```sql
CREATE TABLE price_history (
    id         BIGSERIAL PRIMARY KEY,
    chain      VARCHAR(10) NOT NULL,
    symbol     VARCHAR(10) NOT NULL,      -- ETH/BTC/SOL
    price_usd  TEXT,                      -- 美元价格
    timestamp  BIGINT NOT NULL
);
```

### 7.4 表关系

```
blocks (1) ──── (N) transactions
    │                    │
    │ block.id  ←→  tx.block_id  (外键)
    │
    └── UNIQUE(chain, block_number)  防止重复同步同一区块
```

---

## 8. 前端结构

### 8.1 技术栈

| 技术 | 用途 |
|---|---|
| React 18 | UI 框架 |
| TypeScript | 类型安全的 JavaScript |
| Vite | 构建工具（比 Webpack 快） |
| Tailwind CSS | CSS 工具类（不用写 CSS 文件） |
| React Router | 前端路由 |
| axios | HTTP 请求库 |
| recharts | 图表库（价格曲线） |

### 8.2 组件树

```
<BrowserRouter>                    ← 路由容器
  <ChainProvider>                  ← 链切换上下文（全局共享 chain 状态）
    <Layout>                       ← 页面布局
      ├── <SearchBar />            ← 搜索栏
      ├── <ChainSwitcher />        ← BTC/ETH/SOL 切换按钮
      ├── <PriceCard />            ← 当前价格显示
      └── <Routes>                 ← 路由匹配
          ├── "/" → <BlockList>    ← 首页
          │           ├── <PriceChart />  ← 价格曲线
          │           └── 区块列表表格
          ├── "/blocks/:chain/:number" → <BlockDetail>
          ├── "/tx/:chain/:hash" → <TxDetail>
          └── "/address/:chain/:addr" → <AddressTx>
    </Layout>
  </ChainProvider>
</BrowserRouter>
```

### 8.3 数据获取模式

每个页面组件使用相同的模式：

```tsx
useEffect(() => {
    const fetchData = async () => {
        setLoading(true)
        try {
            const data = await getBlockList(chain, page, pageSize)
            setBlocks(data.blocks)
            setTotal(data.pagination.total)
        } catch (err) {
            console.error('获取失败:', err)
        } finally {
            setLoading(false)
        }
    }
    fetchData()
}, [chain, page])  // 依赖项：chain 或 page 变化时重新执行
```

### 8.4 路由与页面对应关系

| URL | 页面 | 显示内容 |
|---|---|---|
| `/` | BlockList | 价格曲线 + 区块列表 |
| `/blocks/eth/25212557` | BlockDetail | 区块详情 + 区块内交易列表 |
| `/tx/eth/0xabc...` | TxDetail | 交易详情 |
| `/address/eth/0xabc...` | AddressTx | 地址的交易历史 |

---

## 9. Docker 部署架构

### 9.1 9 个容器

```
docker-compose.yaml 定义了 9 个服务：

基础设施（3 个）：
  ├── postgres    (5432)  ← PostgreSQL 数据库
  ├── redis       (6379)  ← Redis 缓存
  └── kafka       (9092)  ← Kafka 消息队列

Go 微服务（5 个）：
  ├── query-api   (8080)  ← 查询 API
  ├── search-api  (8081)  ← 搜索 API
  ├── price-api   (8082)  ← 价格 API
  ├── eth-sync-worker     ← 以太坊同步（无端口暴露）
  └── block-processor     ← 区块处理器（无端口暴露）

前端（1 个）：
  └── web         (3000)  ← nginx + React 静态文件
```

### 9.2 网络通信

```
所有容器都在 blockexplore-net 这个 bridge 网络中
容器之间通过服务名（不是 localhost）互相访问

web → query-api:8080    (nginx 反向代理)
web → search-api:8081   (nginx 反向代理)
web → price-api:8082    (nginx 反向代理)
query-api → postgres:5432
query-api → redis:6379
block-processor → postgres:5432
block-processor → kafka:9092
eth-sync-worker → kafka:9092
price-api → postgres:5432
price-api → redis:6379
```

### 9.3 启动顺序

```
docker compose up -d --build

启动顺序（通过 depends_on + healthcheck 控制）：
  1. postgres（等待 healthy）
  2. redis（等待 healthy）
  3. kafka（等待 healthy）
  4. query-api / search-api / price-api / eth-sync-worker / block-processor（同时启动）
  5. web（等待 query-api 启动）
```

---

## 10. 如何阅读代码

### 10.1 推荐阅读顺序

```
第 1 层：理解数据结构
  └── internal/model/block.go          ← 看看区块长什么样
  └── internal/model/transaction.go    ← 看看交易长什么样
  └── migrations/001_init.sql          ← 看看数据库表结构

第 2 层：理解数据怎么来
  └── internal/client/eth_client.go    ← 怎么从以太坊拿数据
  └── internal/service/sync/eth_sync.go ← 怎么定时同步
  └── internal/mq/producer.go          ← 怎么发到 Kafka

第 3 层：理解数据怎么处理
  └── internal/mq/consumer.go          ← 怎么从 Kafka 消费
  └── internal/service/processor/block_processor.go ← 怎么写入数据库

第 4 层：理解数据怎么查询
  └── internal/repository/block_repo.go ← 怎么查数据库
  └── internal/service/query/query_service.go ← 怎么加缓存
  └── internal/handler/block_handler.go ← 怎么返回 JSON

第 5 层：理解前端怎么展示
  └── web/src/api/client.ts            ← 怎么调 API
  └── web/src/pages/BlockList.tsx      ← 怎么渲染页面
```

### 10.2 关键代码片段导航

| 想了解什么 | 看哪个文件 |
|---|---|
| 区块数据长什么样 | `internal/model/block.go:33` |
| 以太坊 RPC 怎么调用 | `internal/client/eth_client.go:128` |
| 十六进制怎么转十进制 | `internal/client/eth_client.go:325` |
| Kafka 消息格式 | `internal/mq/producer.go:58` |
| 消息怎么写入数据库 | `internal/service/processor/block_processor.go:61` |
| Redis 缓存怎么用 | `pkg/cache/redis.go:57` |
| API 路由怎么注册 | `internal/router/router.go:39` |
| 错误码怎么定义 | `pkg/errcode/errcode.go` |
| 前端怎么调 API | `web/src/api/client.ts` |
| 价格曲线怎么画 | `web/src/components/PriceChart.tsx` |
| 链切换怎么实现 | `web/src/context/ChainContext.tsx` |

### 10.3 Go 语言速查

如果你完全不懂 Go，这几个概念够用了：

```go
// 变量声明
name := "张三"           // 短变量声明，自动推断类型
var age int = 25         // 完整声明

// 结构体（类似 Java 的 class）
type User struct {
    Name string `json:"name"`  // 反引号里是标签，告诉 JSON 库字段名
    Age  int    `json:"age"`
}

// 函数
func add(a int, b int) int {   // 参数类型在后面！
    return a + b
}

// 错误处理
result, err := doSomething()   // Go 函数可以返回两个值
if err != nil {                // 永远要检查 error！
    return err
}

// 切片（动态数组）
nums := []int{1, 2, 3}       // 创建切片
nums = append(nums, 4)        // 追加元素

// 指针
var p *int                    // *int 是指针类型
p = &age                      // & 取地址
fmt.Println(*p)               // * 解引用，获取值

// goroutine（轻量级线程）
go func() {                   // go 关键字启动 goroutine
    fmt.Println("并发执行")
}()

// channel（goroutine 之间的通信管道）
ch := make(chan string)       // 创建 channel
ch <- "hello"                 // 发送数据
msg := <-ch                   // 接收数据

// defer（函数返回前执行）
defer file.Close()            // 确保资源被释放
```

---

## 附录：常见问题

**Q: 为什么 BTC 和 SOL 没有数据？**
A: 目前只启动了 eth-sync-worker。btc-sync-worker 需要本地比特币节点，sol-sync-worker 需要添加到 docker-compose.yaml。

**Q: 为什么价格曲线数据很少？**
A: CoinGecko API 在中国被墙，需要 VPN。价格数据每 30 秒采集一次，运行时间越长数据越多。

**Q: Redis 里存了什么？**
A: 缓存热点数据，减少数据库查询。键格式如 `blocks:eth:1:20`，值是 JSON 序列化的查询结果。

**Q: Kafka 里存了什么？**
A: 区块原始数据。sync-worker 生产消息，block-processor 消费消息。消息格式是 `BlockMessage{chain, block_number, data}`。

**Q: 程序挂了数据会丢吗？**
A: 不会。Kafka 消息持久化，block-processor 重启后会从上次消费的位置继续。数据库数据永久保存。
