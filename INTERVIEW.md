# BlockExplore 面试准备文档

> 面向应届生面试，覆盖项目涉及的全部技术栈。
> 每道题都是真实面试场景重现，附回答要点。

---

## 一、面试官问 Go 语言

### Q1：Go 的指针和 C 的指针有什么区别？为什么 Go 要保留指针？

**回答：**
- Go 保留了指针，但没有指针算术（不能 `p++`），安全得多
- 传值 vs 传指针：Go 函数参数默认是值拷贝，大结构体不传指针会浪费内存
- `*Type` 可以表示 `nil`，用来区分"空的"和"不存在"

**项目里体现：**
```go
// Slot 可以是 nil，表示 Slot 不存在（BTC/ETH 没有这个概念）
Block struct {
    Slot *int64  // *int64 可以为 nil，int64 不行
}
```

### Q2：Go 的 goroutine 和线程有什么区别？

**回答：**
- goroutine 是用户态轻量级线程，初始栈只有 2KB，线程通常 1MB
- goroutine 由 Go 运行时调度（GPM 模型），不是操作系统调度
- 一个程序可以轻松跑百万个 goroutine

**项目里体现：**
```go
// eth_sync.go - 并发获取交易回执
for i := range txs {
    wg.Add(1)
    go func(idx int) {  // 启动 goroutine
        defer wg.Done()
        receipt, err := w.client.GetTransactionReceipt(txs[idx].TxHash)
        // ...
    }(i)
}
wg.Wait()
```

### Q3：channel 是什么？有缓冲和无缓冲的区别？

**回答：**
- channel 是 goroutine 之间通信的管道
- 无缓冲：发送方阻塞直到接收方准备好（同步）
- 有缓冲：缓冲区满之前不阻塞（异步）
- Go 哲学：不要通过共享内存来通信，通过通信来共享内存

**追问：channel 的底层实现？**
- `hchan` 结构体：包含环形队列、等待发送者队列、等待接收者队列
- 发送数据时，如果有等待的接收者就直接传递，不走缓冲区

### Q4：interface{} 和泛型的区别？Go 1.18 泛型做了什么？

**回答：**
- `interface{}` 是运行时多态，有装箱开销，需要类型断言
- 泛型是编译期展开，没有运行时开销，类型安全
- 项目用的 Go 版本没有泛型，用 `interface{}` 处理 JSON 的动态结构

### Q5：defer 的执行顺序？return 和 defer 谁先执行？

**回答：**
- defer 是 LIFO（后进先出）
- return 先赋值，defer 后执行
- 经典陷阱：`defer` 里的 `recover()` 只能在函数内部捕获 panic

### Q6：slice 的扩容机制？

**回答：**
- 容量小于 256 时，扩容为原来的 2 倍
- 容量超过 256 时，扩容约 1.25 倍 + 192（Go 1.18+）
- `append` 可能返回新地址，原来的 slice 可能失效

### Q7：map 是并发安全的吗？

**回答：**
- 不是，并发写 map 会 panic
- 需要用 `sync.Mutex` 或 `sync.Map`
- Go 1.9+ 有 `sync.Map`，适合读多写少

---

## 二、面试官问 TypeScript / Next.js

### Q1：TypeScript 的 type 和 interface 有什么区别？

**回答：**
- `interface` 可以声明合并，`type` 不行
- `type` 可以做联合类型、交叉类型，`interface` 不行
- 对象类型优先用 `interface`，联合类型用 `type`

**项目里体现：**
```typescript
export interface Block {
  id: number
  chain: string
  block_number: number
  // ...
}

export type ChainType = 'eth' | 'btc' | 'sol'  // 联合类型
```

### Q2：Next.js 的 SSR / SSG / CSR 有什么区别？

**回答：**
- SSR（服务端渲染）：每次请求都生成 HTML，SEO 友好
- SSG（静态生成）：构建时生成 HTML，不依赖请求
- CSR（客户端渲染）：浏览器下载 JS 后渲染，首屏慢
- 项目使用 Next.js standalone 模式，动态路由走 SSR

### Q3：React 的 useState 和 useEffect 的执行时机？

**回答：**
- `useState` 是同步的但更新是异步的，React 会批量处理
- `useEffect` 在 DOM 更新后异步执行
- `useEffect` 的依赖数组为空表示只执行一次

### Q4：跨域问题是什么？怎么解决？

**回答：**
- 浏览器同源策略限制不同域的请求
- 解决方案：CORS 头、反向代理、JSONP（不推荐）
- 项目用 Next.js 的 `rewrites` 代理 API 请求，避免跨域

---

## 三、面试官问 PostgreSQL

### Q1：索引为什么快？B+Tree 的结构是怎样的？

**回答：**
- B+Tree 是多路平衡搜索树，所有数据存在叶子节点
- 叶子节点形成有序链表，支持范围查询
- 每个节点对应一个磁盘页（通常 8KB），减少磁盘 IO
- 非叶子节点只存键，一个节点能存更多索引项

### Q2：为什么选 PostgreSQL 而不是 MySQL？

**回答：**
- 对上链数据，`numeric(78,18)` 精度更高（ETH 的 18 位小数）
- PostgreSQL 的 JSONB 支持更好，适合存交易数据
- 并发控制 MVCC 更先进
- 项目需要处理 3 条链的成本，PostgreSQL 的分析函数更强大

### Q3：事务隔离级别有哪些？PostgreSQL 默认是什么？

**回答：**
- 四种级别：读未提交、读已提交、可重复读、可串行化
- PostgreSQL 默认是读已提交
- PostgreSQL 没有读未提交，等同于读已提交
- 用 MVCC 实现，不靠锁

### Q4：Kafka 写入后数据库丢失怎么办？

**回答：**
- 这其实是分布式事务问题
- 项目用的是 At-Least-Once：Kafka 写入成功再写入 PostgreSQL
- 如果 PostgreSQL 写入失败，Kafka 消息不会 ACK，消费者重试
- 数据库用唯一索引防止重复写入

---

## 四、面试官问 Redis

### Q1：Redis 为什么快？

**回答：**
- 纯内存操作，数据结构简单
- 单线程处理网络请求（6.0 后网络 IO 多线程），避免锁竞争
- IO 多路复用（epoll），一个线程处理很多连接
- RESP 协议简单，解析快

### Q2：缓存穿透、击穿、雪崩分别是什么？

**回答：**
- **穿透**：查不存在的数据，绕过缓存打到数据库。解决：布隆过滤器 / 缓存空值
- **击穿**：热点 key 过期，大量请求打到数据库。解决：互斥锁 / 永不过期
- **雪崩**：大量 key 同时过期。解决：过期时间加随机值

### Q3：Redis 的内存淘汰策略？

**回答：**
- `noeviction`：不淘汰，内存满直接报错
- `allkeys-lru`：所有 key 中淘汰最近最少使用的（推荐缓存场景）
- `volatile-lru`：仅设置过期时间的 key 中淘汰
- 项目用 Redis 做缓存，用 `allkeys-lru`

### Q4：Redis 的持久化机制？

**回答：**
- RDB：定时快照，二进制，恢复快，可能丢数据
- AOF：记录每次写操作，恢复慢，丢数据少
- 通常两者结合使用

---

## 五、面试官问 Kafka

### Q1：Kafka 为什么快？顺序写比随机写快多少？

**回答：**
- 顺序写磁盘，不需要寻道，接近内存速度
- Page Cache 缓存磁盘数据，读操作直接命中内存
- 零拷贝（sendfile），数据不经过用户空间直达网卡
- 分区并行，水平扩展
- 顺序写比随机写快约 3 个数量级（几千倍）

### Q2：partition 和 consumer group 的关系？

**回答：**
- 一个 partition 只能被 group 内一个 consumer 消费
- 一个 consumer 可以消费多个 partition
- partition 数量决定了并行消费的上限
- 同一 group 内 consumer 数量不能超过 partition 数量，否则有空闲

### Q3：Kafka 怎么保证消息不丢？

**回答：**
- **Producer 端**：`acks=all`，等所有 replica 确认
- **Broker 端**：多副本（replication factor > 1），`min.insync.replicas` 控制最小同步副本数
- **Consumer 端**：手动提交 offset，处理完再提交

### Q4：消息重复了怎么办？

**回答：**
- Kafka 的语义是 At-Least-Once（可能重复）
- 消费者做幂等：用唯一键（如 tx_hash + block_number）去重
- 或分布式事务（性能差，一般不这么干）

**项目里体现：**
```sql
-- 数据库唯一索引防止重复插入
CREATE UNIQUE INDEX idx_tx_hash ON transactions(tx_hash);
```

### Q5：Kafka 如何支持多 listener？项目为什么配置两个？

**回答：**
- Kafka 支持多 listener，每个 listener 对应一个网络接口
- `KAFKA_LISTENERS` 指定监听的地址和端口
- `KAFKA_ADVERTISED_LISTENERS` 告诉客户端"连接这个地址"
- 项目配置双 listener：INTERNAL 供容器间通信，EXTERNAL 供宿主机访问
- 同一个 Kafka 实例，只是多个入口

---

## 六、面试官问 Docker

### Q1：Docker 和虚拟机有什么区别？

**回答：**
- Docker 共享宿主机内核，虚拟机有独立内核
- Docker 启动秒级，虚拟机分钟级
- Docker 镜像分层存储，虚拟机是完整镜像
- 容器是进程隔离，虚拟机是硬件虚拟化

### Q2：Docker 的镜像分层是什么？为什么每次构建都有 cache？

**回答：**
- Dockerfile 的每个指令生成一个层，层之间共享
- 同一个层的 hash 不变就不重新构建
- 把不常变的（如安装依赖）放在前面，常变的（如 COPY 源码）放在后面
- 这样修改代码不需要重新安装依赖

### Q3：Docker bridge 网络是什么？

**回答：**
- 默认网络驱动，容器之间通过 IP 互相访问
- 内置 DNS，容器名可以当域名用
- 同一 bridge 网络的容器可以直接通信
- 端口映射可以让宿主机访问容器

### Q4：docker-compose 的 depends_on 和 healthcheck 有区别吗？

**回答：**
- `depends_on` 只保证启动顺序，不等待服务就绪
- `healthcheck` 定义服务就绪标准
- 结合使用：`depends_on` + `condition: service_healthy` 等待健康检查通过

**项目里体现：**
```yaml
depends_on:
  postgres:
    condition: service_healthy  # 等待 PG 可以接受连接
```

### Q5：镜像怎么从 1.5GB 减到 50MB？

**回答：**
- 多阶段构建：编译阶段用大镜像，运行阶段只复制二进制
- 基础镜像从 Ubuntu 换成 Alpine（5MB）
- `.dockerignore` 排除 node_modules、.git 等
- 清理包管理器缓存

### Q6：如何区分开发环境和生产环境的 Docker 配置？

**回答：**
- 开发用 `docker-compose.dev.yaml`，只启基础设施
- 生产用 `docker-compose.yaml`，包括所有微服务
- `env_file` 切换不同配置文件（`.env` vs `.env.docker`）
- 开发环境 Kafka 用 `localhost:9092`，Docker 环境用 `kafka:29092`

---

## 七、面试官问项目架构

### Q1：为什么分成 Handler → Service → Repository 三层？

**回答：**
- **隔离变化**：换数据库只改 Repository，换 HTTP 框架只改 Handler
- **可测试**：每层可以单独 mock 测试
- **职责清晰**：Handler 处理 HTTP，Service 处理业务，Repository 处理 SQL
- 反面教材：把所有逻辑写在 Handler 里，改一行要动一整坨

### Q2：cmd/ 目录下为什么有 7 个 main.go？不能写一起吗？

**回答：**
- 每个程序职责不同：查数据、同步区块、处理消息
- 独立部署：哪个负载高就多开几个实例
- 独立升级：改同步逻辑不影响查询 API
- 这就是微服务：一个仓库，多个可部署单元

### Q3：Kafka 在项目里起什么作用？为什么不直接写数据库？

**回答：**
- **解耦**：同步和入库是两个独立步骤
- **削峰**：大量区块同时到达时，Kafka 当缓冲区
- **异步**：同步 Worker 不等待入库完成，继续拉下一个区块
- 直接写数据库：同步 Worker 会被慢查询拖慢

### Q4：Redis 缓存什么？为什么不全部走数据库？

**回答：**
- 缓存热点数据：最新区块列表、当前价格
- 减少数据库压力：1000 个用户同时访问首页，只查一次数据库
- Cache-Aside 模式：先查缓存 → 未命中查数据库 → 写入缓存

### Q5：数据从区块链到前端经历了什么？

**回答：**
```
1. eth-sync-worker: 调用 RPC → 拉取区块 → 发送到 Kafka
2. block-processor: 消费 Kafka → 解析数据 → 写入 PostgreSQL
3. query-api: 接收 HTTP 请求 → 查 Redis 缓存 → 查 PostgreSQL → 返回 JSON
4. Next.js: 通过代理请求 API → 渲染页面 → 返回浏览器
```

### Q6：怎么保证 BlockCypher API 限流不崩？

**回答：**
- Go 的 `time.Ticker` 控制同步间隔
- 指数退避重试：1s → 2s → 4s
- 失败只记日志，不崩溃，等下一个周期重试

### Q7：这个项目怎么能撑住高并发？瓶颈在哪？

**回答：**
- **当前瓶颈**：外部 API 限流（CoinGecko 免费版限制严格）
- **数据库**：加索引 + 读写分离
- **缓存**：Redis 扛读压力
- **Kafka**：扛写压力，削峰填谷
- **水平扩展**：query-api 可以加实例，Kafka consumer 可以加分区

---

## 八、面试官问计算机基础

### Q1：HTTP 和 HTTPS 的区别？TLS 握手过程？

**回答：**
- HTTP 明文传输，HTTPS 经过 TLS 加密
- TLS 握手：客户端 Hello → 服务端 Hello + 证书 → 密钥交换 → 加密通信
- HTTPS 用的是混合加密：非对称加密交换密钥，对称加密传输数据

### Q2：进程和线程的区别？

**回答：**
- 进程是资源分配的最小单位，线程是 CPU 调度的最小单位
- 进程间内存隔离，线程间共享内存
- 线程切换开销小，进程切换需要切换地址空间
- Goroutine 是更轻量的协程

### Q3：TCP 和 UDP 的区别？TCP 为什么可靠？

**回答：**
- TCP 面向连接，UDP 无连接
- TCP 保证顺序（序列号 + 确认应答 + 重传）
- TCP 有流量控制（滑动窗口）和拥塞控制（慢启动）
- UDP 适合实时场景，丢几帧无所谓（视频通话、游戏）

### Q4：三次握手为什么不是两次？

**回答：**
- 两次握手：服务端无法确认客户端收到了自己的 SYN-ACK
- 如果客户端第一次 SYN 延迟到达，两次握手会导致服务端建立一个过期连接
- 三次握手：双方都确认自己发和收没问题

### Q5：数据库索引为什么用 B+Tree 而不用红黑树？

**回答：**
- 数据库存储在磁盘，瓶颈是磁盘 IO 次数
- B+Tree 是多叉树，高度远低于二叉树，磁盘 IO 次数少
- 叶子节点形成链表，范围查询友好
- 红黑树每个节点存一个值，B+Tree 每个节点存多个值

### Q6：缓存和数据库一致性怎么保证？

**回答：**
- **Cache-Aside**（项目用的）：读的时候写缓存，写的时候删缓存
- **Write-Through**：写数据库的同时写缓存
- **延迟双删**：先删缓存 → 写数据库 → 等一会再删一次缓存
- 最终一致性：允许短暂不一致，最终会一致

### Q7：什么是死锁？怎么避免？

**回答：**
- 两个或以上进程互相等待对方释放资源
- 四个必要条件：互斥、持有并等待、不可剥夺、循环等待
- 破坏其中一个就能避免死锁
- 常见做法：统一加锁顺序、设置超时时间

### Q8：同步和异步、阻塞和非阻塞的区别？

**回答：**
- 同步/异步 关注的是调用方式：同步等结果是"一直在那等着"，异步等结果是"我先去做别的，有结果通知我"
- 阻塞/非阻塞 关注的是等待状态：阻塞是"只能干等"，非阻塞是"等的时候能干别的"
- 组合：同步阻塞（最蠢的）、同步非阻塞（轮询）、异步阻塞（少见）、异步非阻塞（最高效）
- 异步非阻塞 = IO 多路复用，两个事件可以独立推进

**项目里体现：**
```go
// 同步阻塞：发了 HTTP 请求，等 30 秒才返回
resp, err := c.httpClient.Post(url, "application/json", body)

// 并发提高效率：10 个 goroutine 同时发送请求
for i := range txs {
    go func(idx int) {  // 并发请求交易回执
        receipt, err := w.client.GetTransactionReceipt(txs[idx].TxHash)
    }(i)
}
```

---

## 九、面试官的刁难问题

### Q1：这个项目有什么缺点？

**回答（诚实但有改进方案）：**
- **缺乏单元测试**：应该给 client、service、repository 各层写 test
- **配置管理可以更好**：现在用 viper + .env，未来可以上配置中心（etcd / consul）
- **没有监控告警**：应该接入 Prometheus + Grafana
- **错误处理不够精细**：很多地方直接传 error，应该定义业务错误码

### Q2：如果让你重新设计，你会怎么做？

**回答：**
- 用 `protobuf` 定义 Kafka 消息格式，而不是 JSON（更小更快）
- 引入 `OpenTelemetry` 做分布式追踪
- 用 gRPC 替代 REST 做服务间通信
- 把 SQL 查询换成更复杂的分析型查询，做更多数据统计

### Q3：你在这个项目里最有成就感的是什么？

**回答（仅供参考，改成自己的话）：**
- 理解了三层架构的设计哲学，知道为什么要分离关注点
- 搞懂了微服务之间的通信方式（HTTP 同步 + Kafka 异步），Kafka 双 listener 配置
- 用最少的改动适配了多链（ETH/BTC/SOL），设计上预留了扩展空间
- 把一个正经项目从零跑通，理解了开发环境和生产环境的完整流程

---

## 十、面试问题速查表

| 技术 | 必问八股 | 项目相关 |
|---|---|---|
| Go | goroutine、channel、defer、slice、map、interface | 为什么分三层、Kafka 消息体设计、并发获取回执 |
| Next.js/TS | SSR 原理、useEffect、type vs interface | 静态导出方案、API 代理、搜索栏路由 |
| PostgreSQL | 索引、事务隔离、MVCC、B+Tree | numeric(78,18) 为什么、多链数据模型 |
| Redis | 快的原因、缓存三兄弟、持久化 | Cache-Aside 模式、缓存什么数据 |
| Kafka | 快的原因、partition、不丢/不重 | 削峰解耦、consumer 重试、双 listener |
| Docker | 与 VM 区别、分层、bridge 网络 | 多阶段构建、dev vs prod 环境、env 覆盖 |
| 计算机基础 | TCP/UDP、三次握手、死锁、同步/异步 | 项目里的同步异步体现 |
