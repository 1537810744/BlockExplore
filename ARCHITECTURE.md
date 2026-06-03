# BlockExplore 项目分层架构

## 分层全景

```
cmd/                          ← 每个程序的"组装车间"（7 个 main.go）
  query-api/main.go           ← 组装 HTTP 请求处理链条
  eth-sync-worker/main.go     ← 组装 ETH 区块同步链条
  btc-sync-worker/main.go     ← 组装 BTC 区块同步链条
  sol-sync-worker/main.go     ← 组装 SOL 区块同步链条
  block-processor/main.go     ← 组装 Kafka 消费→入库链条
  price-api/main.go           ← 组装价格查询链条
  search-api/main.go          ← 组装搜索链条

internal/                     ← 所有可复用的"零件"
  handler/                    ← 第 1 层：HTTP 参数解析 + JSON 返回
  middleware/                 ← HTTP 拦截器（限流、跨域、日志）
  service/                    ← 第 2 层：业务逻辑
    query/query_service.go    ← 查询 + 缓存策略
    sync/eth_sync.go          ← ETH 区块同步调度
    processor/block_processor.go ← 区块解析入库
    price/price_service.go    ← 价格处理
  repository/                 ← 第 3 层：SQL 语句
  client/                     ← 外部 API 调用（ETH RPC, BTC API, SOL RPC）
  mq/                         ← Kafka 消息队列封装
  model/                      ← 数据结构定义（Block, Transaction 等）
  router/router.go            ← URL 路由注册 + 中间件链
  config/                     ← 配置读取
```

---

## 数据流 1：区块链数据 → 数据库（ETH 为例）

```
[以太坊节点 RPC]
    │  eth_getBlockByNumber
    ▼
① internal/client/eth_client.go:42  GetBlockByNumber()
    │  返回 *model.Block + []model.Transaction
    ▼
② internal/service/sync/eth_sync.go:63  sync()
    │  调用 client 拿到数据，封装成 BlockMessage
    ▼
③ internal/mq/producer.go:95  Send()
    │  序列化为 JSON，写入 Kafka Topic "block.raw.eth"
    ▼
④ internal/mq/consumer.go:90  Consume()
    │  从 Kafka 读取 JSON，反序列化为 BlockMessage
    ▼
⑤ internal/service/processor/block_processor.go:61  Handle()
    │  解析 JSON → model.Block + []model.Transaction
    ▼
⑥ internal/repository/block_repo.go:67  CreateSingle(&block)
   internal/repository/tx_repo.go      Create(transactions)
    │  GORM 生成 INSERT SQL，写入 PostgreSQL
    ▼
[PostgreSQL]
```

---

## 数据流 2：前端 HTTP 请求 → JSON 响应（查区块列表）

```
[浏览器] GET /api/v1/blocks?chain=eth&page=1&page_size=20
    │
    ▼
① web/nginx.conf:20  location /api/
    │  proxy_pass http://query-api:8080  （只是转发，不做任何处理）
    ▼
② internal/router/router.go:44  blocks.GET("", blockHandler.GetBlockList)
    │  匹配 URL，交给 BlockHandler
    ▼
③ internal/middleware/  （依次执行）
    request_id.go → cors.go → gin.Recovery() → ratelimit.go
    │
    ▼
④ internal/handler/block_handler.go:53  GetBlockList()
    │  c.DefaultQuery("chain", "eth")     ← 解析 HTTP 参数
    │  c.DefaultQuery("page", "1")
    │  c.DefaultQuery("page_size", "20")
    │  调用 h.queryService.GetBlockList(chain, page, pageSize)
    ▼
⑤ internal/service/query/query_service.go:93  GetBlockList()
    │  拼缓存 Key: "blocks:eth:1:20"
    │  先查 Redis（100 行）
    │  未命中 → 调 s.blockRepo.GetList(chain, page, pageSize)
    ▼
⑥ internal/repository/block_repo.go:110  GetList()
    │  r.db.Model(&model.Block{}).Where("chain = ?", chain).Count(&total)
    │  r.db.Where(...).Order("block_number DESC").Offset(...).Limit(...).Find(&blocks)
    │  执行: SELECT * FROM blocks WHERE chain='eth' ORDER BY block_number DESC LIMIT 20 OFFSET 0
    ▼
⑦ 结果原路返回：
    Repository → Service → Handler → c.JSON(200, result) → nginx → 浏览器
```

---

## 为什么分这么多层？每一层的边界

### 三层架构：Handler → Service → Repository

| 层 | 知道的事 | 不知道的事 |
|---|---|---|
| **Handler** | HTTP 参数名、JSON 格式、状态码 | 有没有缓存、SQL 怎么写 |
| **Service** | 缓存策略、业务规则、数据校验 | URL 长什么样、SQL 怎么写 |
| **Repository** | SQL 语句、表名、索引 | HTTP 参数、有没有缓存、谁在调用 |

### 分层的核心价值：隔离变化

假设把 PostgreSQL 换成 MySQL，只需要改 **1 个文件**：

```
internal/repository/block_repo.go   ← 改 GORM 驱动，SQL 方言调整
```

Handler 和 Service **一行都不用改**。因为：
- Handler 不知道数据从哪来的，它只是调 `queryService.GetBlockList()`
- Service 不知道 SQL 怎么写，它只是调 `blockRepo.GetList()`

同理，想把 REST API 改成 gRPC，只需要改 Handler 层，Service 和 Repository 不变。

---

## cmd/ 的角色：组装车间

**cmd/ 不实现任何逻辑，它只做一件事：组装零件。**

```go
// cmd/query-api/main.go:97-108
blockRepo := repository.NewBlockRepo(db)       // 创建零件
queryService := query.NewQueryService(blockRepo, txRepo, redisClient)  // 组装
blockHandler := handler.NewBlockHandler(queryService)  // 再组装
r := router.Setup(blockHandler, txHandler, ...) // 最终装配成 HTTP 服务
```

每个 cmd/ 程序是**不同用途的组装方案**，用的零件各不相同：

| 程序 | 用了哪些零件 |
|---|---|
| `query-api` | Handler + Service + Repository + Redis |
| `eth-sync-worker` | Client + Sync Service + Kafka Producer |
| `block-processor` | Kafka Consumer + Processor Service + Repository |
| `price-api` | Handler + Price Service + Repository + 外部 API |

同一个 `BlockRepo`（零件），被 **query-api** 用来查，被 **block-processor** 用来写。零件是同一个，组装出来的机器不同。

---

## 各层的职责边界

### Handler 层（`internal/handler/`）
- 解析 HTTP 请求参数（路径参数、查询参数、请求体）
- 参数校验（页码范围、必填项等）
- 调用 Service 层
- 构建统一格式的 JSON 响应（errcode.Success / errcode.Error）
- **不包含任何业务逻辑和数据库操作**

### Middleware 层（`internal/middleware/`）
- Gin 框架的拦截器链
- 执行顺序：RequestID → CORS → Recovery → RateLimiter
- 每个请求都会依次经过这些中间件

### Service 层（`internal/service/`）
- 封装业务逻辑（缓存策略、数据组装、外部调用编排）
- 调用 Repository 层进行数据读写
- **不知道 HTTP 细节，不解析请求参数**

### Repository 层（`internal/repository/`）
- 封装所有 SQL 操作（GORM）
- 提供数据访问方法：Create、GetByChainAndNumber、GetList 等
- **不包含业务逻辑，不知道谁在调用自己**

### Client 层（`internal/client/`）
- 封装外部 API 调用（ETH JSON-RPC、BTC Mempool.space API、SOL RPC）
- 处理 HTTP 请求、超时、错误重试
- 返回项目内部的 model 结构体（屏蔽外部 API 的数据格式差异）

### MQ 层（`internal/mq/`）
- 封装 Kafka 生产者和消费者
- Producer：序列化消息并发送到指定 Topic
- Consumer：从 Topic 读取消息，反序列化后调用业务处理函数

### Model 层（`internal/model/`）
- 纯数据结构定义（GORM 模型）
- 对应数据库表结构
- 不含任何方法逻辑（贫血模型）

### Router 层（`internal/router/`）
- 定义 URL 路径到 Handler 方法的映射
- 注册全局中间件
- 创建路由组（/api/v1/blocks、/api/v1/transactions 等）
