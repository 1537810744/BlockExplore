# BlockExplore Docker 容器化详解

## 一、打包的东西在哪里？

Docker 镜像不是普通文件，存储在 Docker 内部。用命令查看：

```
docker images | grep blockexplore
```

构建"说明书"在这里：

```
项目根目录/
├── Dockerfile                  ← Go 微服务的打包说明书（所有后端共用）
├── web/Dockerfile              ← 前端的打包说明书
└── docker-compose.yaml         ← 总体编排文件（7 个微服务如何拼在一起）
```

### 一个 Dockerfile 打包出 7 个程序

```
Dockerfile:27-33
  RUN go build -o /app/bin/query-api ./cmd/query-api/ && \
      go build -o /app/bin/search-api ./cmd/search-api/ && \
      go build -o /app/bin/price-api ./cmd/price-api/ && \
      go build -o /app/bin/eth-sync-worker ./cmd/eth-sync-worker/ && \
      go build -o /app/bin/btc-sync-worker ./cmd/btc-sync-worker/ && \
      go build -o /app/bin/sol-sync-worker ./cmd/sol-sync-worker/ && \
      go build -o /app/bin/block-processor ./cmd/block-processor/
```

所有 7 个程序打进了**同一个镜像**。运行时用 `command:` 指定启动哪个：

```
docker-compose.yaml:83   →  command: ["/app/query-api"]         ← 查询 API
docker-compose.yaml:107  →  command: ["/app/search-api"]        ← 搜索 API
docker-compose.yaml:129  →  command: ["/app/block-processor"]   ← 区块处理器
docker-compose.yaml:153  →  command: ["/app/price-api"]         ← 价格 API
docker-compose.yaml:173  →  command: ["/app/eth-sync-worker"]   ← ETH 同步
docker-compose.yaml:189  →  command: ["/app/btc-sync-worker"]   ← BTC 同步
docker-compose.yaml:210  →  command: ["/app/sol-sync-worker"]   ← SOL 同步
```

**7 个容器，同一个镜像，只是启动命令不同。**

### 构建命令

```bash
docker compose build          # 构建所有镜像
docker compose up -d          # 启动所有服务
docker compose down           # 停止所有服务
```

### 多阶段构建（Dockerfile 结构）

```
阶段 1: golang:1.21-alpine  →  编译 Go 源码成二进制文件
阶段 2: alpine:3.19         →  只复制二进制文件，最终镜像体积小
```

阶段 1 的 golang 编译工具链不会进入最终镜像，最终镜像只包含 Alpine Linux + 二进制文件 + ca-certificates。

---

## 二、容器化之前 vs 容器化之后：地址怎么变？

### 本地开发（Go 程序跑在 Windows 上，基础设施在 Docker 里）

```
Go 程序 ──localhost:5432──→ PostgreSQL（Docker 容器，端口映射到宿主机）
Go 程序 ──localhost:6379──→ Redis（Docker 容器，端口映射到宿主机）
Go 程序 ──localhost:9092──→ Kafka（Docker 容器，端口映射到宿主机）
```

Docker 把容器端口映射到宿主机：

```
docker-compose.yaml:17  →  ports: "5432:5432"   ← 宿主机:容器
docker-compose.yaml:33  →  ports: "6379:6379"
docker-compose.yaml:48  →  ports: "9092:9092"
```

### 完全容器化（所有程序都在 Docker bridge 网络内）

```
容器 query-api ──postgres:5432──→ 容器 postgres
容器 query-api ──redis:6379────→ 容器 redis
容器 eth-sync  ──kafka:9092───→ 容器 kafka
```

`"postgres"` 不是 localhost，而是 Docker Compose 的**服务名**。Docker 内部 DNS 自动把服务名解析成容器的内网 IP。

### 具体到代码：只改了一个文件

对比两个配置文件的变化：

| 配置项 | `.env`（本地开发） | `.env.docker`（容器内） |
|---|---|---|
| `DB_HOST` | `localhost` | `postgres` |
| `REDIS_HOST` | `localhost` | `redis` |
| `KAFKA_BROKERS` | `localhost:9092` | `kafka:9092` |

**代码一行都不用改**，因为代码不写死地址：

```
config.go:194  →  cfg.DB.Host = viper.GetString("DB_HOST")
config.go:204  →  cfg.Redis.Host = viper.GetString("REDIS_HOST")
config.go:211  →  cfg.Kafka.Brokers = viper.GetStringSlice("KAFKA_BROKERS")
```

然后自动拼成连接字符串：

```
config.go:74-76  →  DSN() = "host=postgres port=5432 ..."     ← 容器内
                      DSN() = "host=localhost port=5432 ..."   ← 本地
config.go:92-94  →  Addr() = "redis:6379"                     ← 容器内
                      Addr() = "localhost:6379"                ← 本地
```

---

## 三、如果代码里写死 8080 呢？

**端口写死不影响 bridge，影响 bridge 的是 HOST 地址。**

需要理解两个层次：

```
docker-compose.yaml:92  →  ports: "8080:8080"
                                    ^^^^   ^^^^
                                   宿主机  容器内
```

- `8080:8080` 的意思是：把宿主机的 8080 端口，映射到容器内的 8080 端口
- 容器内程序监听 `:8080`，这个 8080 是多少都行
- 容器之间互相访问，走的是容器内部端口 + Docker 服务名，跟宿主机端口映射无关

```
开发时写死 "localhost:5432"  ← 问题在 HOST，不在端口
容器内写死 "postgres:5432"   ← 修正的是 HOST，端口没变
```

**HOST 不能写死，所以用 env 变成可赋值的。端口写死也没关系，但用 env 更灵活。**

---

## 四、完整的服务发现机制

### Docker Bridge 网络

```
docker-compose.yaml:240  →  networks:
                              blockexplore-net:
                                driver: bridge
```

所有服务都在同一个 `blockexplore-net` 桥接网络内，Docker 内部 DNS 自动把**服务名**解析成容器 IP。

### 服务之间的调用关系

```
query-api ──postgres:5432──→ postgres          (.env.docker:11 DB_HOST=postgres)
query-api ──redis:6379────→ redis              (.env.docker:21 REDIS_HOST=redis)

eth-sync-worker ──kafka:9092──→ kafka          (.env.docker:28 KAFKA_BROKERS=kafka:9092)
eth-sync-worker ──外部 HTTPS──→ eth.drpc.org   (.env.docker:35 ETH_RPC_URL=https://eth.drpc.org)

block-processor ──kafka:9092──→ kafka           (.env.docker:28)
block-processor ──postgres:5432──→ postgres     (.env.docker:11)

search-api ──postgres:5432──→ postgres          (.env.docker:11)

price-api ──postgres:5432──→ postgres           (.env.docker:11)
price-api ──redis:6379────→ redis               (.env.docker:21)

web（nginx）──query-api:8080──→ query-api        (nginx.conf:35 proxy_pass http://query-api:8080)
web（nginx）──search-api:8081──→ search-api      (nginx.conf:21 proxy_pass http://search-api:8081)
web（nginx）──price-api:8082──→ price-api        (nginx.conf:27 proxy_pass http://price-api:8082)
```

### 前端不直接访问后端

```
浏览器 ──→ nginx（web 容器，端口 3000）──→ query-api:8080
                                         search-api:8081
                                         price-api:8082
```

### 反向代理中的服务名

```
nginx.conf:21  →  proxy_pass http://search-api:8081;
nginx.conf:27  →  proxy_pass http://price-api:8082;
nginx.conf:35  →  proxy_pass http://query-api:8080;
```

`search-api`、`price-api`、`query-api` 都是 docker-compose.yaml 中定义的服务名。

---

## 五、env 参数的三层覆盖机制

```
config.go:157-169  →  func Load() *Config {
    viper.SetConfigName(".env")          // 第 1 层：读 .env 文件
    viper.AutomaticEnv()                 // 第 2 层：环境变量可覆盖文件中的值
    setDefaults()                        // 第 3 层：都没设就用默认值
}
```

**优先级：环境变量 > .env 文件 > 默认值**

### 在 Docker 中的实际生效路径

```
docker-compose.yaml:154  →  env_file:
                              - .env.docker          ← 第 1 层：读文件注入环境变量
                            environment:
                              HTTP_PROXY=             ← 第 2 层：额外环境变量覆盖
```

容器启动时，docker-compose 把 `.env.docker` 的内容注入容器，`environment:` 里的值可以覆盖文件中的值。

### 默认值（保底）

```
config.go:259  →  viper.SetDefault("DB_HOST", "postgres")
config.go:270  →  viper.SetDefault("REDIS_HOST", "redis")
config.go:277  →  viper.SetDefault("KAFKA_BROKERS", []string{"kafka:9092"})
config.go:297  →  viper.SetDefault("QUERY_API_PORT", 8080)
config.go:298  →  viper.SetDefault("SEARCH_API_PORT", 8081)
config.go:299  →  viper.SetDefault("PRICE_API_PORT", 8082)
```

默认值直接写的就是 Docker 服务名，所以即使没有 env 文件，在 Docker 环境里也能正常运行。

---

## 六、需要代理的服务 vs 不需要代理的服务

中国网络环境下，访问境外 API 需要代理。但不是所有服务都需要：

### 不需要代理（清除 HTTP_PROXY）

```
docker-compose.yaml
  86-90  →  query-api         ← 只查数据库+Redis，纯内网，清空代理
  113-115 → search-api        ← 只查数据库，纯内网，清空代理
  132-136 → block-processor   ← 只读 Kafka+写数据库，纯内网，清空代理
  192-196 → btc-sync-worker   ← Mempool.space 在中国可直连
```

### 需要代理

```
docker-compose.yaml
  148-165 →  price-api        ← CoinGecko API 被墙
  167-181 →  eth-sync-worker  ← 以太坊 RPC 被墙
  204-218 →  sol-sync-worker  ← Solana RPC 被墙
```

代理通过 Docker Desktop 全局配置注入（`c:\Users\admin\.docker\config.json`），需要代理的服务不加 `HTTP_PROXY=` 覆盖，让 Docker 的全局代理生效。

---

## 七、关键转变总结

| | 容器化之前 | 容器化之后 | 变的文件 |
|---|---|---|---|
| DB 地址 | `localhost:5432` | `postgres:5432` | `.env` → `.env.docker` |
| Redis 地址 | `localhost:6379` | `redis:6379` | `.env` → `.env.docker` |
| Kafka 地址 | `localhost:9092` | `kafka:9092` | `.env` → `.env.docker` |
| Go 程序在哪跑 | Windows 直接跑 | 容器内跑 | 多了 `Dockerfile` |
| 前端在哪跑 | `npm run dev` | nginx 容器 | 多了 `web/Dockerfile` |
| 服务编排 | 手动一个个启动 | `docker compose up -d` 一键全起 | 多了 `docker-compose.yaml` |
| 代码改动 | — | **0 行** | 只改了 env 文件 |

**核心原理：代码不写死地址，全部从环境变量读取。容器化只需要换一个 env 文件，代码 0 改动。**
