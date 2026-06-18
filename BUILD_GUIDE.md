# BlockExplore 从 0 到 1 手动构建指南（Windows 11 完整版）

> **适用对象**：Windows 11 新手，有基本的命令行概念。
> **最终目标**：从一个空文件夹开始，一步步搭建出一个完整的多链区块链浏览器——能编译、能跑测试、有 CI/CD、能 Docker 一键部署、对外可访问。
> **预计时间**：跟着做一遍大约需要 4-8 小时（取决于网络速度）。
> **本指南的特性**：每一段代码、每一条命令都与最终仓库**逐字对应**。把项目全删了，照着这份指南也能 100% 复现。

---

## 序章：一个软件到底是怎么从 0 到 1 做出来的

你现在的状态我太懂了：打开编辑器，想到哪写到哪，跑不通就 `print` 满天飞，改完忘了删 `print`，最后得到一个"到处漏风的垃圾"。这份指南的正文是 33 章的逐步操作，但在你动手之前，我想先用**大白话 + 这个项目的真实代码**，把"一个正经软件是怎么从无到有做出来的"整个流程按时间顺序讲一遍。不讲为什么对、为什么错，只讲**真实开发过程里到底发生了什么**。这一节不分章节，从头到尾顺着读。

### 第一步：想清楚要做什么（需求）

盖房子不能边盖边想"要不要加个厨房"，盖到二楼才发现一楼没留厕所。软件也一样。先花十分钟写下来：这个东西要干嘛、给谁用、有哪些功能。

本项目我就写了一句话：**做一个多链区块链浏览器，支持 ETH/BTC/SOL 三条链，能查区块、交易、地址，有价格图表，界面像 Etherscan 那样的深色风格**。就这一句，决定了后面所有代码的方向。如果跳过这步，你会写到一半突然"要不要也加个 NFT 浏览器"，然后项目永远做不完。

### 第二步：画架构图（设计）

施工图。不画就开干，最后水管走进电线管里。我用 ASCII 画了一张数据流向图：

```
区块链节点 → sync-worker 拉取 → Kafka → block-processor 入库 → PostgreSQL
                                                              ↑
                                          query-api 查 ← Redis 缓存
                                                              ↓
                                                    Next.js 前端 → 浏览器
```

这张图告诉你：数据从哪来、经过哪些环节、最后到谁手里。每个环节就是一个程序。这张图一画出来，我就知道要做 7 个程序（3 个 sync-worker + 1 个 processor + 3 个 api）+ 1 个前端。**不是想出来的，是画出来的**。

然后定了个铁律叫"三层架构"：`Handler（接 HTTP 请求）→ Service（业务逻辑）→ Repository（数据库操作）`。为什么分三层后面代码里你会体会到——一句话：换数据库时只改 Repository，Handler 和 Service 一行不用动。

### 第三步：选技术栈

选建材。木头房还是钢筋混凝土房，提前定好。本项目选了：Go（后端语言）、Gin（Web 框架）、GORM（数据库操作）、PostgreSQL（数据库）、Redis（缓存）、Kafka（消息队列）、Next.js（前端）、Docker（部署）。

你不用纠结为什么选这些。重点是：**定下来之后就别中途换**。写到一半把 PostgreSQL 换成 MySQL，整个 Repository 层都要重写。

### 第四步：建空骨架 + 立刻存档（Git）

盖楼先搭脚手架。我做的第一件事不是写代码，是 `mkdir` 把所有目录建出来：

```bash
mkdir -p cmd/query-api internal/config internal/model ...
```

空目录，一个文件没有。然后**立刻**做两件事，这是新手最常跳过的：

**第一件：`git init` + 写 `.gitignore`。** Git 是什么？就是你打游戏时的**存档点**。每 `git commit` 一次就是存一个档，改坏了随时 `git checkout` 读档回来。新手不存档，改坏了疯狂 Ctrl+Z 撤不回来，只能对着屏幕哭。

`.gitignore` 是"哪些东西不存档"的清单。比如 `.env`（里面有数据库密码，不能传到公开仓库）、`node_modules/`（依赖包，几百兆，能重新下载不用存）、`*.exe`（编译产物）。

```gitignore
.env
node_modules/
*.exe
bin/
```

这一步花五分钟，省掉以后无数麻烦。

**第二件：第一个 commit。** 哪怕只有空目录和 `.gitignore`，也立刻 `git commit -m "chore: init scaffold"`。从此你有了第一个存档点。

### 第五步：先通水电（基础设施）

盖房子先通水电，不然砌砖时没水拌水泥。软件的"水电"就是数据库、缓存、消息队列这些。我用 Docker 一条命令把它们全拉起来：

```bash
docker compose -f docker-compose.dev.yaml up -d
```

这条命令启动了 PostgreSQL（存数据）、Redis（缓存）、Kafka（消息队列）。然后**验证它们真的通了**（不是启动就完事）：

```bash
docker exec blockexplore-dev-postgres psql -U blockexplore -c "SELECT 1;"
# 看到 ?column? = 1，才算通
```

新手跳过验证，直接去写代码，结果代码连不上数据库，不知道是代码错还是数据库没起来，排查两小时。**每一步都要当场验证**。

### 第六步：从地基往上写（分层依赖）

这是新手最大的毛病：想到哪写到哪。今天写前端，明天写后端，后天回来改数据库。正经做法是**从最底层、不依赖别人的部分开始，一层层往上**。

本项目的真实书写顺序就是从下到上：

| 顺序 | 包 | 依赖谁 | 为什么这个顺序 |
|------|-----|--------|--------------|
| 1 | config | 谁都不依赖 | 最底层，读配置 |
| 2 | logger | 谁都不依赖 | 其他包都要打日志 |
| 3 | cache | config | 需要读 Redis 地址 |
| 4 | errcode | 谁都不依赖 | 纯定义常量 |
| 5 | model | 谁都不依赖 | 纯定义数据结构 |
| 6 | repository | model | 要操作 model |
| 7 | mq | config | 要读 Kafka 地址 |
| 8 | client | model | 要返回 model |
| 9 | sync | client + mq | 调 client 拉数据，发 mq |
| 10 | processor | mq + repository | 消费 mq，写 repository |
| 11 | query | repository + cache | 查 repository，用 cache |
| 12 | handler | service | 接 HTTP，调 service |
| 13 | router | handler | 把 URL 接到 handler |
| 14 | cmd | 所有 | 组装所有零件，启动 |

为什么不能先写 handler（最上层）？因为 handler 要调 service，service 还没写，编译就报错。**从下往上写，每写一层，下面的都已经是好的，编译能过**。

### 第七步：每砌一块砖就测平不平（编译验证）

这是和"漏风垃圾"最大的区别。新手写完 500 行才编译，报 20 个错，不知道从哪改。我的做法：**每写完一个包（大概 50-150 行），立刻编译**：

```bash
go build ./internal/config/   # 写完 config 就编译
# 没输出 = 没错。继续写下一个
go build ./internal/model/    # 写完 model 再编译
```

这样错误永远只有一两个，而且一定在刚写的那 50 行里，三秒就能找到。**频繁编译 = 错误局部化 = 改起来轻松**。

### 第八步：跑不通怎么办——调试，不是乱 print

这是新手第二大病：跑不通就 `fmt.Println("到这里了")` 满天撒，改完忘记删，上线后日志里全是"到这里了"。正经调试是这样的：

**第一步：读错误信息。** Go 的错误信息很友好，会告诉你哪个文件哪一行出了什么问题。比如：

```
dial tcp: lookup postgres: no such host
```

这一行就告诉你：连不上 `postgres` 这个主机名。为什么？因为你在本地跑，本地没有 `postgres` 这个域名，应该用 `localhost`。去 `.env` 把 `DB_HOST=postgres` 改成 `DB_HOST=localhost`。**不用任何 print，错误信息自己就说了答案**。

**第二步：看日志。** 我们用 zap 日志库，每条日志带时间、级别、文件行号：

```json
{"level":"info","timestamp":"2026-06-18T20:45:19","caller":"config/config.go:237","msg":"已加载配置文件: .../.env"}
```

看到"已加载配置文件"就知道配置读到了。看到 `[配置] DB_HOST=localhost` 就知道值对不对。**日志是程序的体检报告，不是 print**。

**第三步：缩小范围。** 真不知道哪错了，注释掉一半代码，看错误还在不在。在的话错误在被注释掉的那半，不在就在没注释的那半。对半切两次就定位到具体几行了。比 print 快十倍。

**第四步：IDE 断点。** VS Code 里点行号左边设个红点，按 F5 调试，程序跑到那行会停，你能看每个变量的值。这是最专业的调试方式。

本项目里真实发生过一个 bug：query-api 启动后，访问价格接口就崩溃。我看日志看到 panic 在 `router.go` 里调 `priceHandler.GetCurrentPrice`，因为 `priceHandler` 是 `nil`。**不用 print，看 panic 的堆栈就定位到了**。修复就是 `cmd/query-api/main.go` 里把 `nil` 换成真实的 `PriceService` 实例。

### 第九步：能跑了，怎么保证以后改不坏——写测试

新手觉得："我手动跑一遍通了不就行了？" 问题在于：你以后改一行代码，要手动把所有功能再跑一遍吗？100 个功能呢？你肯定会漏。**测试就是替你自动跑一遍的脚本**。

本项目 `pkg/errcode/errcode_test.go` 里写了这样的测试：

```go
func TestSuccess(t *testing.T) {
	resp := Success("hello", "req-1")
	assert.Equal(t, 200, resp.Code)         // 期望 Code 是 200
	assert.Equal(t, "success", resp.Message) // 期望 Message 是 success
}
```

跑一下：

```bash
go test -v ./pkg/errcode/
# --- PASS: TestSuccess (0.00s)
# PASS
```

以后我改了 `Success` 函数的实现，再跑这个测试，如果还 PASS 就说明没改坏，FAIL 就说明改坏了。**测试让你敢重构**。

写测试的关键：**测一个函数时，把它依赖的东西换成假的（mock）**。比如测 `QueryService` 时，不连真实数据库，而是传一个假的 `BlockRepository`（接口实现），它直接返回写死的数据。这样测试又快又稳定，不依赖外部环境。

本项目把 `QueryService` 和 `BlockProcessor` 的依赖改成了**接口**（`BlockRepository`、`Cacher`、`BlockWriter`），就是为了让测试能传 mock：

```go
// 真实运行时传真的 repo
svc := query.NewQueryService(blockRepo, txRepo, redisClient)
// 测试时传假的 repo
svc := query.NewQueryService(&mockBlockRepo{...}, &mockTxRepo{}, nil)
```

这是"用接口不用具体类型"的真正价值——不是为了装高级，是为了能测。

### 第十步：怎么知道测试有没有漏——覆盖率

你写了 10 个测试，但代码里有 50 个函数，剩下 40 个没测过，就是漏的。**覆盖率**就是告诉你"你的测试跑过了多少代码"：

```bash
go test -cover ./...
# blockexplore/internal/model    coverage: 100.0%
# blockexplore/internal/handler  coverage: 32.3%
```

100% 表示全测过了。32% 表示 handler 里一大半代码没测过——那就是潜在的炸弹。还能生成 HTML 报告，绿色是测过的，红色是没测的：

```bash
go tool cover -html=coverage.out -o coverage.html
```

**不用追求 100%**（有些代码很难测），但红色那一大片就是要补的。

### 第十一步：存档点的高级用法——分支

之前说 `git commit` 是存档。现在说**分支**。分支是什么？就是**在草稿纸上乱写，写好了再抄到正式本子上**。

新手一直在 `main`（正式本子）上改，改到一半想试试另一个方案，又不敢改怕回不去。正确做法：

```bash
git checkout -b feat/add-price-chart   # 复制 main 到一张新草稿纸
# ...在这里随便改...
git commit -m "feat: add price chart"  # 草稿纸上的改动存档
git checkout main                       # 切回正式本子
git merge feat/add-price-chart          # 把草稿纸内容抄到正式本子
git branch -d feat/add-price-chart      # 扔掉草稿纸
```

如果草稿纸上写砸了？直接 `git checkout main` 走人，草稿纸扔掉，正式本子一点没污染。**分支让你敢乱改**。

### 第十二步：合并的麻烦——冲突

两张草稿纸都改了同一行，合并时 Git 不知道听谁的，就叫冲突。比如 main 上是 `price = 100`，你的分支上是 `price = 200`，合并时会出现：

```
<<<<<<< HEAD
price = 100
=======
price = 200
>>>>>>> feat/add-price-chart
```

你手动编辑这一段，决定留哪个（或写成新的），删掉那些 `<<<` `===` `>>>` 标记，再 `git add` + `git commit`。**冲突不可怕，就是让你当裁判**。一个人开发时很少冲突，多人改同一个文件才会经常遇到。

### 第十三步：门口安检——CI（持续集成）

你提交代码前手动跑 `go build` 和 `go test`，觉得通了就提交。但你哪天赶时间偷懒不跑呢？CI 就是**机器每次你提交都自动跑一遍，不通过不让合并**。

本项目 `.github/workflows/ci.yml` 定义了四道安检门：

```
你 push 代码
  → 门1 Lint（代码风格检查，错别字、没用的变量）
  → 门2 Test（跑所有单元测试 + 覆盖率）
  → 门3 Build（编译全部 7 个服务）
  → 门4 Integration（启动真实数据库，跑集成测试）
全过 → ✅ 允许合并
任一失败 → ❌ 通知你修
```

CI 跑在 GitHub 的服务器上，你提交后自动开始，不用你管。**它不会偷懒，每次都跑全套**。

### 第十四步：排版校对——代码质量工具

交稿前要校对错别字、排版。代码也一样：变量声明了没用、错误没处理、缩进不统一——这些"代码臭味"由 `golangci-lint` 检查。还能配 `pre-commit hook`：每次 `git commit` 前自动跑一遍检查，不通过就拦住你：

```bash
git config core.hooksPath scripts/hooks
# 以后每次 commit，会自动跑 gofmt + go vet + go test，全过才让提交
```

**让机器替你守规矩**。

### 第十五步：交房——部署

代码在你电脑上跑通了，但别人没法访问你的电脑。**部署**就是把程序放到一台一直开着的机器上，让别人通过网络访问。

本项目用 Docker 把 11 个服务（3 个基础设施 + 7 个 Go 程序 + 1 个前端）打包成容器，一条命令全起来：

```bash
docker compose up -d --build
```

别人访问 `http://你的服务器IP:3000` 就能用你的浏览器了。

### 第十六步：挂牌营业——让外人访问

`localhost:3000` 只有你自己能访问（localhost = 本机）。让全世界访问，有三种办法：

1. **内网穿透**（最快，临时演示）：用 cloudflared 一行命令，把你本机的 3000 端口暴露成一个公网网址，发给别人。
2. **云服务器**（长期）：买台服务器，把代码传上去 `docker compose up`，配域名 + HTTPS。
3. **镜像推到 Registry**（最专业）：把镜像推到 Docker Hub，服务器拉下来跑，适合多次部署和自动部署。

本项目第 33 章会详细讲这三种。

---

### 把全过程串起来

上面这十六步就是"一个软件从 0 到 1"的真实时序。**和正文 33 章的对应关系**：

- 第 0-1 章 = 步骤 1-4（想清楚、画图、选型、建骨架+Git）
- 第 2-3 章 = 步骤 5（通水电：Docker 起基础设施 + 建表）
- 第 4-21 章 = 步骤 6-8（从地基往上写 + 每层编译验证 + 调试）
- 第 22-24 章 = 步骤 15（前端 + Docker 部署 + 端到端验证）
- 第 25 章 = 步骤 8 的故障排查手册
- 第 26 章 = 步骤 11-12（Git 分支 + Conventional Commits）
- 第 27-30 章 = 步骤 9-10（单元测试、集成测试、benchmark、覆盖率）
- 第 31 章 = 步骤 13（CI）
- 第 32 章 = 步骤 14（代码质量工具）
- 第 33 章 = 步骤 16（对外部署）

**你现在明白这份指南的结构了吗？** 它不是"想到哪写到哪"的笔记，它是**按真实开发时序组织**的工程流程。你照着做一遍，做出来的不是一个"漏风的垃圾"，而是一个能编译、有测试、有 CI、能部署、别人能访问的**正经软件**。

序章结束。下面是"阅读约定"和正式的 33 章逐步操作。新手建议先把序章记住（尤其"从下往上写"和"每步验证"这两条），再开始动手。

---

## 阅读约定

1. **所有命令在 Git Bash 中运行**（PowerShell 会在需要时单独标注）。
2. 代码块顶部会写明文件路径，例如 `// internal/config/config.go`，照着创建即可。
3. 每章结尾有"编译检查"或"验证"步骤，**不要跳过**——这是确认你没写错的关键。
4. 带有 `<details>` 折叠块的是补充说明，新手建议展开看一遍。

---

## 目录

- [第 0 章：开始之前 —— 检查你的电脑环境](#第-0-章开始之前--检查你的电脑环境)
- [第 1 章：创建项目骨架与 Git 初始化](#第-1-章创建项目骨架与-git-初始化)
- [第 2 章：基础设施 —— Docker 启动 PostgreSQL、Redis、Kafka](#第-2-章基础设施--docker-启动-postgresqlrediskafka)
- [第 3 章：数据库设计 —— 建表 SQL](#第-3-章数据库设计--建表-sql)
- [第 4 章：Go 模块初始化与依赖安装](#第-4-章go-模块初始化与依赖安装)
- [第 5 章：配置管理 —— config 包](#第-5-章配置管理--config-包)
- [第 6 章：日志工具 —— logger 包](#第-6-章日志工具--logger-包)
- [第 7 章：Redis 缓存工具 —— cache 包](#第-7-章redis-缓存工具--cache-包)
- [第 8 章：错误码工具 —— errcode 包](#第-8-章错误码工具--errcode-包)
- [第 9 章：数据模型层 —— model 包](#第-9-章数据模型层--model-包)
- [第 10 章：数据访问层 —— repository 包](#第-10-章数据访问层--repository-包)
- [第 11 章：Kafka 消息队列 —— mq 包](#第-11-章kafka-消息队列--mq-包)
- [第 12 章：区块链客户端 —— client 包](#第-12-章区块链客户端--client-包)
- [第 13 章：区块同步服务 —— sync 包](#第-13-章区块同步服务--sync-包)
- [第 14 章：区块处理服务 —— processor 包](#第-14-章区块处理服务--processor-包)
- [第 15 章：价格服务 —— price 包](#第-15-章价格服务--price-包)
- [第 16 章：查询服务 —— query 包](#第-16-章查询服务--query-包)
- [第 17 章：HTTP 处理层 —— handler 包](#第-17-章http-处理层--handler-包)
- [第 18 章：中间件 —— middleware 包](#第-18-章中间件--middleware-包)
- [第 19 章：路由注册 —— router 包](#第-19-章路由注册--router-包)
- [第 20 章：七个微服务入口 —— cmd/](#第-20-章七个微服务入口--cmd)
- [第 21 章：编译验证 —— 让代码跑起来](#第-21-章编译验证--让代码跑起来)
- [第 22 章：Next.js 前端 —— web/](#第-22-章nextjs-前端--web)
- [第 23 章：Docker 容器化](#第-23-章docker-容器化)
- [第 24 章：完整运行与端到端验证](#第-24-章完整运行与端到端验证)
- [第 25 章：故障排查手册](#第-25-章故障排查手册)
- [第 26 章：Git 分支管理与 Conventional Commits](#第-26-章git-分支管理与-conventional-commits)
- [第 27 章：Go 单元测试 —— 用代码验证代码](#第-27-章go-单元测试--用代码验证代码)
- [第 28 章：集成测试 —— 端到端验证](#第-28-章集成测试--端到端验证)
- [第 29 章：性能基准测试 —— benchmark](#第-29-章性能基准测试--benchmark)
- [第 30 章：测试覆盖率](#第-30-章测试覆盖率)
- [第 31 章：CI/CD —— GitHub Actions 自动化流水线](#第-31-章cicd--github-actions-自动化流水线)
- [第 32 章：代码质量工具](#第-32-章代码质量工具)
- [第 33 章：对外部署 —— 让全世界访问你的浏览器](#第-33-章对外部署--让全世界访问你的浏览器)
- [总结：从零到一的完整路径](#总结从零到一的完整路径)

---

## 第 0 章：开始之前 —— 检查你的电脑环境

### 0.1 打开 Git Bash

按 `Win` 键，输入 `git bash`，点击打开。你应该看到类似这样的窗口：

```
admin@DESKTOP-XXX MINGW64 ~
$
```

> **为什么用 Git Bash？** 它提供 Unix 风格命令（`ls`、`mkdir`、`touch`），和服务器环境一致，命令可以直接复制粘贴。PowerShell 的 `touch`、`&&` 语法不同，会报错。

<details>
<summary>如果你坚持用 PowerShell</summary>

每个会话开始时先运行：
```powershell
function touch($f) { New-Item -ItemType File -Path $f -Force | Out-Null }
```
`mkdir -p X/Y/Z` 改为 `New-Item -ItemType Directory -Force -Path X/Y/Z`。
**强烈建议直接用 Git Bash，省去所有麻烦。**

</details>

### 0.2 检查 Go

```bash
go version
```
**应该看到**：`go version go1.21.x windows/amd64`。

<details>
<summary>找不到 go？</summary>

1. 访问 https://go.dev/dl/
2. 下载 `go1.21.x.windows-amd64.msi`
3. 双击安装，保持默认路径 `C:\Program Files\Go\`
4. **关闭 Git Bash 重新打开**（刷新 PATH）
5. 再次 `go version`

</details>

### 0.3 检查 Node.js

```bash
node --version
```
**应该看到**：`v20.x.x`。访问 https://nodejs.org/ 下载 LTS 版安装。

### 0.4 检查 Docker Desktop

```bash
docker --version
docker compose version
```
**应该看到**：`Docker version 25.x+` 和 `Docker Compose version v2.x+`。

<details>
<summary>找不到 docker？</summary>

1. 访问 https://www.docker.com/products/docker-desktop/
2. 下载 Docker Desktop for Windows，双击安装，**需要重启电脑**
3. 重启后等右下角鲸鱼图标稳定（不再转圈）
4. 在 Docker Desktop 设置里确认 WSL2 后端已启用

</details>

### 0.5 检查 Git

```bash
git --version
```
**应该看到**：`git version 2.x.x`。打开了 Git Bash 就说明 Git 已装好。

### 0.6 （可选）准备代理

区块链节点（Ethereum/Solana RPC）和 CoinGecko 在中国大陆需要代理。如果你用 Clash/V2Ray，确认它的本地端口（常见 `7890`）。Docker 构建会用到。

**如果你有代理**：保持它在 7890 端口监听。
**如果你没有代理**：第 12-15 章的链上同步会失败，但项目骨架、测试、Docker 部署仍能完成（数据库为空，前端显示"暂无数据"）。可以先跳过链上同步部分。

### 0.7 创建项目文件夹

```bash
cd ~/Desktop
mkdir BlockExplore
cd BlockExplore
pwd
# 应该输出: /c/Users/你的用户名/Desktop/BlockExplore
```

**现在你的 BlockExplore 文件夹是完全空的。** 下面一步一步来。

---

## 第 1 章：创建项目骨架与 Git 初始化

### 1.1 创建所有目录

```bash
mkdir -p cmd/query-api
mkdir -p cmd/search-api
mkdir -p cmd/price-api
mkdir -p cmd/eth-sync-worker
mkdir -p cmd/btc-sync-worker
mkdir -p cmd/sol-sync-worker
mkdir -p cmd/block-processor
mkdir -p internal/config
mkdir -p internal/model
mkdir -p internal/repository
mkdir -p internal/mq
mkdir -p internal/client
mkdir -p internal/handler
mkdir -p internal/router
mkdir -p internal/middleware
mkdir -p internal/service/query
mkdir -p internal/service/sync
mkdir -p internal/service/processor
mkdir -p internal/service/price
mkdir -p pkg/cache
mkdir -p pkg/logger
mkdir -p pkg/errcode
mkdir -p migrations
mkdir -p scripts/hooks
mkdir -p .github/workflows
```

### 1.2 初始化 Git 仓库

```bash
git init
```

### 1.3 创建 .gitignore

```bash
touch .gitignore
```

写入：

```gitignore
# Go 编译产物
*.exe
*.exe~
*.dll
*.so
*.dylib
/bin/

# Node.js
node_modules/
web/node_modules/
web/.next/

# 测试产物
*.test
*.out
coverage.out
coverage.html

# Go 模块缓存
vendor/

# IDE 配置
.idea/
.vscode/
*.swp
*.swo
*~

# 环境配置文件（包含敏感信息）
.env

# Docker 数据卷
postgres_data/
redis_data/
kafka_data/

# 操作系统文件
.DS_Store
Thumbs.db

# 日志文件
*.log
```

> **为什么 `.env` 要被忽略？** `.env` 里会有你的 RPC API Key，不能提交到公开仓库。我们之后会创建 `.env.example` 作为模板提交。

### 1.4 验证目录结构

```bash
ls -R
```

你应该看到 20+ 个空目录。这就是项目的骨架。

---

## 第 2 章：基础设施 —— Docker 启动 PostgreSQL、Redis、Kafka

### 2.1 理解开发环境的架构

开发阶段**只需要 Docker 启动三个基础服务**：PostgreSQL、Redis、Kafka。Go 程序和前端在本地直接运行，方便调试。

```
你的 Windows 电脑
├── Docker Desktop
│   ├── PostgreSQL (端口 5432)
│   ├── Redis (端口 6379)
│   └── Kafka (端口 9092)
├── Go 程序 (本地运行，通过 localhost 连接上述服务)
└── Next.js 前端 (本地运行，端口 3000)
```

### 2.2 创建开发环境 docker-compose 文件

```bash
touch docker-compose.dev.yaml
```

写入 `docker-compose.dev.yaml`：

```yaml
# ============================================================
# BlockExplore 开发环境 Docker Compose
# 只启动基础设施，Go 程序在本地运行
# ============================================================
# 使用方式：
#   docker compose -f docker-compose.dev.yaml up -d
#   或
#   docker compose -f docker-compose.dev.yaml up -d postgres redis kafka
#
# 端口冲突时覆盖（例如 Windows Hyper-V 保留了 5432）：
#   POSTGRES_PORT=5280 docker compose -f docker-compose.dev.yaml up -d
# ============================================================

services:
  postgres:
    image: postgres:latest
    container_name: blockexplore-dev-postgres
    environment:
      POSTGRES_USER: blockexplore
      POSTGRES_PASSWORD: blockexplore123
      POSTGRES_DB: blockexplore
      PGDATA: /var/lib/postgresql/data/pgdata
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    volumes:
      - postgres_dev_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d  # 自动执行 migrations 下的 SQL
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U blockexplore"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:latest
    container_name: blockexplore-dev-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_dev_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  kafka:
    image: apache/kafka:latest
    container_name: blockexplore-dev-kafka
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      # 开发环境：单 listener，localhost 直连
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"  # 允许自动创建 topic
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      KAFKA_MESSAGE_MAX_BYTES: 10485760       # 最大消息 10MB
      KAFKA_REPLICA_FETCH_MAX_BYTES: 10485760
      CLUSTER_ID: MkU3OEVBNTcwNTJENDM2Qk
    volumes:
      - kafka_dev_data:/var/lib/kafka/data
    healthcheck:
      test: ["CMD-SHELL", "/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 > /dev/null 2>&1"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 30s
    restart: unless-stopped

volumes:
  postgres_dev_data:
  redis_dev_data:
  kafka_dev_data:
```

**关键点说明**：
- `./migrations:/docker-entrypoint-initdb.d`：PostgreSQL 首次启动时会自动执行 `migrations/` 下的 SQL 文件建表。
- `KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"`：sync worker 第一次发消息时自动创建 topic，否则会报 "Unknown Topic"。
- `${POSTGRES_PORT:-5432}`：端口可被环境变量覆盖。Windows Hyper-V 经常保留 5432 端口段，遇到冲突时用 `POSTGRES_PORT=5280` 即可。

### 2.3 启动基础设施

```bash
docker compose -f docker-compose.dev.yaml up -d
```

**应该看到**：
```
[+] Running 3/3
 ✔ Container blockexplore-dev-postgres  Started
 ✔ Container blockexplore-dev-redis     Started
 ✔ Container blockexplore-dev-kafka     Started
```

<details>
<summary>如果报 "port is already allocated" 或 5432 被占用</summary>

Windows Hyper-V 会随机保留一段端口。检查：
```bash
netsh interface ipv4 show excludedportrange protocol=tcp
```
如果 5432 落在保留段内，用备用端口启动：
```bash
POSTGRES_PORT=5280 docker compose -f docker-compose.dev.yaml up -d
```
后续所有连接 PostgreSQL 的地方都用 5280（本指南会在需要时提醒）。

</details>

### 2.4 验证服务健康

```bash
docker compose -f docker-compose.dev.yaml ps
```

**应该看到**三个服务都是 `healthy`：
```
NAME                         STATUS
blockexplore-dev-postgres    Up (healthy)
blockexplore-dev-redis       Up (healthy)
blockexplore-dev-kafka       Up (healthy)
```

Kafka 启动较慢（30-60 秒），如果显示 `starting`，等一会儿再查。

### 2.5 测试连接

```bash
# PostgreSQL
docker exec blockexplore-dev-postgres psql -U blockexplore -c "SELECT 1;"
# 应该看到: ?column? = 1

# Redis
docker exec blockexplore-dev-redis redis-cli ping
# 应该看到: PONG

# Kafka
docker exec blockexplore-dev-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list
# 应该看到: __consumer_offsets（自动创建的内部 topic）
```

三个都通了，基础设施 OK。

---

## 第 3 章：数据库设计 —— 建表 SQL

### 3.1 创建数据库迁移文件

```bash
touch migrations/001_init.sql
```

写入 `migrations/001_init.sql`（与项目实际使用的完全一致）：

```sql
-- ============================================================
-- BlockExplore 数据库初始化脚本
-- 创建所有表结构和索引
-- ============================================================

-- 区块表：存储各链的区块信息
CREATE TABLE IF NOT EXISTS blocks (
    id              BIGSERIAL PRIMARY KEY,              -- 主键 ID（自增）
    chain           VARCHAR(10) NOT NULL,               -- 链标识: eth/btc/sol
    block_number    BIGINT NOT NULL,                    -- 区块高度
    block_hash      VARCHAR(128) NOT NULL,              -- 区块哈希
    parent_hash     VARCHAR(128),                       -- 父区块哈希
    timestamp       BIGINT NOT NULL,                    -- 出块时间（Unix 时间戳）
    tx_count        INT DEFAULT 0,                      -- 区块内交易数量
    gas_used        TEXT,                               -- 已消耗 Gas（ETH/SOL）
    gas_limit       TEXT,                               -- Gas 上限（ETH）
    size_bytes      INT,                                -- 区块大小（字节，BTC）
    difficulty      TEXT,                               -- 难度值（BTC）
    slot            BIGINT,                             -- 槽位号（SOL）
    created_at      TIMESTAMP DEFAULT NOW(),            -- 记录创建时间
    UNIQUE(chain, block_number),                        -- 同一条链的区块高度唯一
    UNIQUE(chain, block_hash)                           -- 同一条链的区块哈希唯一
);

-- 区块表索引：加速按链和区块高度的查询
CREATE INDEX IF NOT EXISTS idx_blocks_chain_number ON blocks(chain, block_number DESC);
-- 区块表索引：加速按时间的查询
CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(chain, timestamp DESC);

-- 交易表：存储各链的交易信息
CREATE TABLE IF NOT EXISTS transactions (
    id              BIGSERIAL PRIMARY KEY,              -- 主键 ID（自增）
    chain           VARCHAR(10) NOT NULL,               -- 链标识: eth/btc/sol
    tx_hash         VARCHAR(128) NOT NULL,              -- 交易哈希
    block_number    BIGINT NOT NULL,                    -- 所在区块高度
    block_id        BIGINT REFERENCES blocks(id),       -- 关联的区块表 ID（外键）
    from_addr       VARCHAR(128),                       -- 发送方地址
    to_addr         VARCHAR(128),                       -- 接收方地址
    value           TEXT,                               -- 转账金额
    gas_price       TEXT,                               -- Gas 价格（ETH）
    gas_used        TEXT,                               -- 实际消耗 Gas（ETH）
    gas_limit       TEXT,                               -- Gas 上限（ETH）
    nonce           BIGINT,                             -- 交易序号（ETH）
    input_data      TEXT,                               -- 调用数据（ETH calldata）
    status          SMALLINT DEFAULT 1,                 -- 交易状态：1=成功 0=失败
    timestamp       BIGINT NOT NULL,                    -- 交易时间（Unix 时间戳）
    created_at      TIMESTAMP DEFAULT NOW(),            -- 记录创建时间
    UNIQUE(chain, tx_hash)                              -- 同一条链的交易哈希唯一
);

-- 交易表索引：加速按区块查询
CREATE INDEX IF NOT EXISTS idx_tx_block ON transactions(block_id);
-- 交易表索引：加速按发送方地址查询
CREATE INDEX IF NOT EXISTS idx_tx_from ON transactions(from_addr);
-- 交易表索引：加速按接收方地址查询
CREATE INDEX IF NOT EXISTS idx_tx_to ON transactions(to_addr);
-- 交易表索引：加速按交易哈希查询
CREATE INDEX IF NOT EXISTS idx_tx_hash ON transactions(chain, tx_hash);

-- 地址表：记录地址的交易统计信息
CREATE TABLE IF NOT EXISTS addresses (
    id              BIGSERIAL PRIMARY KEY,              -- 主键 ID
    chain           VARCHAR(10) NOT NULL,               -- 链标识: eth/btc/sol
    address         VARCHAR(128) NOT NULL,              -- 区块链地址
    balance         NUMERIC(78,18),                     -- 当前余额
    tx_count        BIGINT DEFAULT 0,                   -- 交易总数
    first_seen_at   BIGINT,                             -- 首次交易时间
    last_seen_at    BIGINT,                             -- 最近交易时间
    created_at      TIMESTAMP DEFAULT NOW(),            -- 记录创建时间
    updated_at      TIMESTAMP DEFAULT NOW(),            -- 记录更新时间
    UNIQUE(chain, address)                              -- 同一条链的地址唯一
);

-- 价格历史表：记录各链原生代币的历史价格
CREATE TABLE IF NOT EXISTS price_history (
    id              BIGSERIAL PRIMARY KEY,              -- 主键 ID
    chain           VARCHAR(10) NOT NULL,               -- 链标识: eth/btc/sol
    symbol          VARCHAR(10) NOT NULL,               -- 代币符号: ETH/BTC/SOL
    price_usd       TEXT,                               -- 美元价格
    timestamp       BIGINT NOT NULL,                    -- 价格时间（Unix 时间戳）
    created_at      TIMESTAMP DEFAULT NOW()             -- 记录创建时间
);

-- 价格历史表索引：加速按链和时间的查询
CREATE INDEX IF NOT EXISTS idx_price_chain_time ON price_history(chain, timestamp DESC);
```

**四张表的作用**：
| 表 | 存什么 |
|---|---|
| `blocks` | 每个区块的元数据（高度、哈希、时间、Gas 等） |
| `transactions` | 每笔交易（哈希、发送方、接收方、金额、状态） |
| `addresses` | 地址统计（余额、交易数、首末交易时间） |
| `price_history` | 代币价格历史（用于价格曲线图） |

### 3.2 让数据库执行 SQL

PostgreSQL 容器**首次启动**时会自动执行 `migrations/` 下的 SQL（因为 compose 挂载了 `./migrations:/docker-entrypoint-initdb.d`）。

如果你在第 2 章已经启动了 PostgreSQL，需要重建让它执行 SQL：

```bash
# -v 删除数据卷（清空数据库），开发阶段没关系
docker compose -f docker-compose.dev.yaml down -v
docker compose -f docker-compose.dev.yaml up -d

# 等几秒后验证表是否创建
docker exec blockexplore-dev-postgres psql -U blockexplore -c "\dt"
```

**应该看到**四张表：
```
 Schema |     Name      | Type  |    Owner
--------+---------------+-------+-------------
 public | addresses     | table | blockexplore
 public | blocks        | table | blockexplore
 public | price_history | table | blockexplore
 public | transactions  | table | blockexplore
```

> **新手常见误操作**：改了 `001_init.sql` 后，必须 `down -v` 再 `up -d` 才生效。`-v` 会清空所有数据。

---

## 第 4 章：Go 模块初始化与依赖安装

### 4.1 初始化 Go module

```bash
cd ~/Desktop/BlockExplore
go mod init blockexplore
```
**应该看到**：`go: creating new go.mod: module blockexplore`

### 4.2 安装第三方依赖

```bash
# Web 框架：Gin
go get github.com/gin-gonic/gin@v1.9.1

# ORM 框架：GORM + PostgreSQL 驱动
go get gorm.io/gorm@v1.25.5
go get gorm.io/driver/postgres@v1.5.4

# Redis 客户端
go get github.com/go-redis/redis/v8@v8.11.5

# Kafka 客户端
go get github.com/segmentio/kafka-go@v0.4.47

# 配置管理
go get github.com/spf13/viper@v1.18.2

# 高性能日志
go get go.uber.org/zap@v1.27.0

# UUID 生成（请求 ID）
go get github.com/google/uuid@v1.6.0

# 定时任务（price-api 用 cron 同步价格）
go get github.com/robfig/cron/v3@v3.0.1

# 测试断言库（单元测试用）
go get github.com/stretchr/testify@v1.9.0
```

每条命令应看到 `go: added xxx`。如果卡住，可能是网络问题：
```bash
# 国内用户可设 GOPROXY 加速
go env -w GOPROXY=https://goproxy.cn,direct
```

### 4.3 验证 go.mod

```bash
cat go.mod
```

你应该看到类似：
```
module blockexplore

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
    github.com/google/uuid v1.6.0
    github.com/robfig/cron/v3 v3.0.1
    github.com/segmentio/kafka-go v0.4.47
    github.com/spf13/viper v1.18.2
    github.com/stretchr/testify v1.9.0
    go.uber.org/zap v1.27.0
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)
```

---

## 第 5 章：配置管理 —— config 包

### 5.1 创建配置文件

```bash
touch .env.example
touch .env
```

写入 `.env.example`（这是模板，会被提交到 Git）：

```bash
# ============================================================
# BlockExplore 配置文件模板
# 复制此文件为 .env 并修改对应的值
# ============================================================

# ---------- 应用配置 ----------
APP_NAME=blockexplore
APP_ENV=development
APP_DEBUG=true

# ---------- PostgreSQL 数据库配置 ----------
# Docker 内部连接使用 host: postgres（容器服务名）
# 本地连接使用 host: localhost
DB_HOST=localhost
DB_PORT=5432
DB_USER=blockexplore
DB_PASSWORD=blockexplore123
DB_NAME=blockexplore
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=10

# ---------- Redis 缓存配置 ----------
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=100

# ---------- Kafka 消息队列配置 ----------
KAFKA_BROKERS=localhost:9092
KAFKA_ETH_TOPIC=block.raw.eth
KAFKA_BTC_TOPIC=block.raw.btc
KAFKA_SOL_TOPIC=block.raw.sol
KAFKA_CONSUMER_GROUP=block-processor-group

# ---------- 以太坊 RPC 配置 ----------
# 免费注册: https://www.alchemy.com/ 或 https://infura.io/
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY
ETH_SYNC_INTERVAL=12

# ---------- 比特币 RPC 配置 ----------
# 用 BlockCypher API（代码里硬编码了这个 URL，此处仅占位）
BTC_RPC_URL=https://api.blockcypher.com/v1/btc/main
BTC_RPC_USER=bitcoin
BTC_RPC_PASSWORD=bitcoin123
BTC_SYNC_INTERVAL=600

# ---------- Solana RPC 配置 ----------
SOL_RPC_URL=https://api.mainnet-beta.solana.com
SOL_SYNC_INTERVAL=1

# ---------- 价格 API 配置 ----------
PRICE_API_URL=https://api.coingecko.com/api/v3
PRICE_SYNC_INTERVAL=30

# ---------- API 服务端口配置 ----------
QUERY_API_PORT=8080
SEARCH_API_PORT=8081
PRICE_API_PORT=8082

# ---------- 日志配置 ----------
LOG_LEVEL=info
LOG_FORMAT=json
```

然后把 `.env.example` 复制为 `.env`（本地开发用，不提交到 Git）：

```bash
cp .env.example .env
```

打开 `.env`，把 `ETH_RPC_URL` 里的 `YOUR_API_KEY` 替换为你自己的 Alchemy API Key（去 https://www.alchemy.com/ 免费注册，创建 Ethereum Mainnet App，复制 HTTPS URL）。如果你用了备用 PostgreSQL 端口，把 `DB_PORT` 改成对应值（如 `5280`）。

### 5.2 编写 config.go

```bash
touch internal/config/config.go
```

写入 `internal/config/config.go`（完整 384 行，与项目实际代码逐字一致）：

```go
// ============================================================
// Package config 提供统一的配置管理功能
// ============================================================
// 该包负责从 .env 文件和环境变量中读取配置，供整个应用使用。
//
// 配置优先级：环境变量 > .env 文件 > 默认值
//
// 使用的第三方库：
//   - github.com/spf13/viper：配置管理库，支持多种配置格式（JSON、YAML、ENV 等）
//
// Go 语言基础知识:
//   - package：Go 的包管理机制，每个目录是一个包
//   - struct：结构体，类似于 Java 的 class，用于定义数据结构
//   - mapstructure：结构体标签，用于配置映射
//   - func (c DBConfig) DSN()：方法定义，c 是接收者（receiver），类似于 this
//   - viper.GetString()：从配置中获取字符串值
// ============================================================
package config

import (
	"fmt"  // 格式化字符串
	"log"  // 标准日志库
	"os"   // 操作系统功能
	"path/filepath" // 文件路径处理

	"github.com/spf13/viper" // 配置管理库，由 spf13 开发
)

// ============================================================
// Config 应用总配置结构体
// ============================================================
// 包含所有子配置：应用、数据库、Redis、Kafka、各链配置等
// mapstructure:",squash" 表示将嵌套结构体的字段展平到父结构体中
type Config struct {
	App    AppConfig    `mapstructure:",squash"` // 应用基础配置
	DB     DBConfig     `mapstructure:",squash"` // 数据库配置
	Redis  RedisConfig  `mapstructure:",squash"` // Redis 缓存配置
	Kafka  KafkaConfig  `mapstructure:",squash"` // Kafka 消息队列配置
	ETH    ChainConfig  `mapstructure:",squash"` // 以太坊链配置
	BTC    BTCConfig    `mapstructure:",squash"` // 比特币链配置
	SOL    ChainConfig  `mapstructure:",squash"` // Solana 链配置
	Price  PriceConfig  `mapstructure:",squash"` // 价格 API 配置
	Server ServerConfig `mapstructure:",squash"` // API 服务端口配置
	Log    LogConfig    `mapstructure:",squash"` // 日志配置
	Proxy  ProxyConfig  `mapstructure:",squash"` // 代理配置
}

// ============================================================
// ProxyConfig 代理配置
// ============================================================
type ProxyConfig struct {
	HTTP  string // HTTP 代理地址
	HTTPS string // HTTPS 代理地址
}

// ============================================================
// AppConfig 应用基础配置
// ============================================================
type AppConfig struct {
	Name  string // 应用名称，例如 "blockexplore"
	Env   string // 运行环境：development（开发）/ production（生产）
	Debug bool   // 是否开启调试模式（true 会输出更多日志）
}

// ============================================================
// DBConfig PostgreSQL 数据库配置
// ============================================================
// PostgreSQL 是一个功能强大的开源关系型数据库
type DBConfig struct {
	Host         string // 数据库主机地址，例如 "localhost" 或 Docker 服务名 "postgres"
	Port         int    // 数据库端口，默认 5432
	User         string // 数据库用户名
	Password     string // 数据库密码
	DBName       string // 数据库名称
	SSLMode      string // SSL 模式："disable"（不加密）/ "require"（要求加密）
	MaxOpenConns int    // 最大打开连接数（并发连接上限）
	MaxIdleConns int    // 最大空闲连接数（空闲时保持的连接数）
}

// DSN 方法返回 PostgreSQL 连接字符串
// DSN = Data Source Name（数据源名称）
// 格式: "host=localhost port=5432 user=xxx password=xxx dbname=xxx sslmode=disable"
// 方法定义语法：func (接收者类型) 方法名(参数) 返回值 { ... }
// 接收者 c 类似于 Java 的 this，表示当前实例
func (c DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// ============================================================
// RedisConfig Redis 缓存配置
// ============================================================
// Redis 是一个高性能的内存数据库，常用于缓存和消息队列
type RedisConfig struct {
	Host     string // Redis 主机地址
	Port     int    // Redis 端口，默认 6379
	Password string // Redis 密码（无密码留空）
	DB       int    // Redis 数据库编号（0-15，共 16 个数据库）
	PoolSize int    // 连接池大小（同时保持的最大连接数）
}

// Addr 方法返回 Redis 地址，格式: "host:port"
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ============================================================
// KafkaConfig Kafka 消息队列配置
// ============================================================
// Kafka 是分布式消息队列，用于在服务之间传递消息
// Topic 是消息的分类，类似于邮件的收件箱
type KafkaConfig struct {
	Brokers       []string // Kafka Broker 地址列表，例如 ["kafka:9092"]
	ETHTopic      string   // 以太坊区块数据 Topic，例如 "block.raw.eth"
	RTCTopic      string   // 比特币区块数据 Topic，例如 "block.raw.btc"
	SOLTopic      string   // Solana 区块数据 Topic，例如 "block.raw.sol"
	ConsumerGroup string   // 消费者组名称，同一组内的消费者共享消息
}

// ============================================================
// ChainConfig 通用链配置（ETH/SOL）
// ============================================================
// 以太坊和 Solana 的配置结构相同，共用这个结构体
type ChainConfig struct {
	RPCURL       string // RPC 节点 URL，例如 "http://localhost:8545"
	SyncInterval int    // 同步间隔（秒），每隔多少秒拉取一次新区块
}

// ============================================================
// BTCConfig 比特币链配置
// ============================================================
// 比特币节点需要额外的用户名密码认证（HTTP Basic Auth）
type BTCConfig struct {
	RPCURL       string // RPC 节点 URL
	RPCUser      string // RPC 用户名
	RPCPassword  string // RPC 密码
	SyncInterval int    // 同步间隔（秒）
}

// ============================================================
// PriceConfig 价格 API 配置
// ============================================================
type PriceConfig struct {
	APIURL       string // 价格 API 地址，例如 "https://api.coingecko.com/api/v3"
	SyncInterval int    // 价格同步间隔（秒）
}

// ============================================================
// ServerConfig API 服务端口配置
// ============================================================
type ServerConfig struct {
	QueryAPIPort  int // 查询 API 端口，默认 8080
	SearchAPIPort int // 搜索 API 端口，默认 8081
	PriceAPIPort  int // 价格 API 端口，默认 8082
}

// ============================================================
// LogConfig 日志配置
// ============================================================
type LogConfig struct {
	Level  string // 日志级别：debug（调试）/ info（信息）/ warn（警告）/ error（错误）
	Format string // 日志格式：json（JSON 格式，适合生产环境）/ console（控制台格式，适合开发）
}

// findProjectRoot 查找项目根目录（包含 go.mod 的目录）
func findProjectRoot() string {
	// 先从当前工作目录查找
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// 再从可执行文件所在目录查找
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "."
}

// ============================================================
// Load 函数：从环境变量和 .env 文件加载配置
// ============================================================
// 优先级：环境变量 > .env 文件 > 默认值
// 返回值：*Config 指针类型，Go 中指针传递避免拷贝整个结构体
func Load() *Config {
	// ---- 设置配置文件名和路径 ----
	viper.SetConfigName(".env") // 配置文件名（不带扩展名）
	viper.SetConfigType("env")  // 配置文件类型为 .env 格式
	viper.AddConfigPath(".")    // 在当前目录查找配置文件
	viper.AddConfigPath("..")   // 也在上级目录查找（用于 cmd 子目录运行时）
	viper.AddConfigPath("../..") // 再上级目录查找

	// 查找项目根目录并添加到配置搜索路径
	if root := findProjectRoot(); root != "." {
		viper.AddConfigPath(root)
	}

	// ---- 允许读取环境变量 ----
	// AutomaticEnv() 会自动读取环境变量，覆盖配置文件中的同名配置
	// 例如：环境变量 DB_HOST 会覆盖 .env 文件中的 DB_HOST
	viper.AutomaticEnv()

	// ---- 读取配置文件 ----
	// 如果文件不存在也不报错，因为有环境变量和默认值兜底
	if err := viper.ReadInConfig(); err != nil {
		// 类型断言：检查错误是否是 ConfigFileNotFoundError 类型
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("警告: 读取配置文件失败: %v", err)
		} else {
			// 打印搜索路径帮助调试
			cwd, _ := os.Getwd()
			log.Printf("警告: 未找到 .env 配置文件，当前工作目录: %s", cwd)
			log.Printf("提示: 请在项目根目录运行程序，或设置环境变量")
		}
	} else {
		log.Printf("已加载配置文件: %s", viper.ConfigFileUsed())
	}

	// ---- 设置所有默认值 ----
	setDefaults()

	// ---- 导出代理配置到环境变量 ----
	// 客户端通过 os.Getenv() 读取代理，需要手动导出
	if httpProxy := viper.GetString("HTTP_PROXY"); httpProxy != "" {
		os.Setenv("HTTP_PROXY", httpProxy)
	}
	if httpsProxy := viper.GetString("HTTPS_PROXY"); httpsProxy != "" {
		os.Setenv("HTTPS_PROXY", httpsProxy)
	}

	// 调试：打印关键配置值
	log.Printf("[配置] DB_HOST=%s, REDIS_HOST=%s, KAFKA_BROKERS=%s",
		viper.GetString("DB_HOST"),
		viper.GetString("REDIS_HOST"),
		viper.GetString("KAFKA_BROKERS"),
	)

	// ---- 解析配置到结构体 ----
	// viper.GetString() 从配置中获取字符串值
	// 如果配置中没有设置，返回默认值
	cfg := &Config{}

	// 应用配置
	cfg.App.Name = viper.GetString("APP_NAME")
	cfg.App.Env = viper.GetString("APP_ENV")
	cfg.App.Debug = viper.GetBool("APP_DEBUG")

	// 数据库配置
	cfg.DB.Host = viper.GetString("DB_HOST")
	cfg.DB.Port = viper.GetInt("DB_PORT")
	cfg.DB.User = viper.GetString("DB_USER")
	cfg.DB.Password = viper.GetString("DB_PASSWORD")
	cfg.DB.DBName = viper.GetString("DB_NAME")
	cfg.DB.SSLMode = viper.GetString("DB_SSLMODE")
	cfg.DB.MaxOpenConns = viper.GetInt("DB_MAX_OPEN_CONNS")
	cfg.DB.MaxIdleConns = viper.GetInt("DB_MAX_IDLE_CONNS")

	// Redis 配置
	cfg.Redis.Host = viper.GetString("REDIS_HOST")
	cfg.Redis.Port = viper.GetInt("REDIS_PORT")
	cfg.Redis.Password = viper.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = viper.GetInt("REDIS_DB")
	cfg.Redis.PoolSize = viper.GetInt("REDIS_POOL_SIZE")

	// Kafka 配置
	cfg.Kafka.Brokers = viper.GetStringSlice("KAFKA_BROKERS") // GetStringSlice 获取字符串切片
	cfg.Kafka.ETHTopic = viper.GetString("KAFKA_ETH_TOPIC")
	cfg.Kafka.RTCTopic = viper.GetString("KAFKA_BTC_TOPIC")
	cfg.Kafka.SOLTopic = viper.GetString("KAFKA_SOL_TOPIC")
	cfg.Kafka.ConsumerGroup = viper.GetString("KAFKA_CONSUMER_GROUP")

	// 以太坊配置
	cfg.ETH.RPCURL = viper.GetString("ETH_RPC_URL")
	cfg.ETH.SyncInterval = viper.GetInt("ETH_SYNC_INTERVAL")

	// 比特币配置
	cfg.BTC.RPCURL = viper.GetString("BTC_RPC_URL")
	cfg.BTC.RPCUser = viper.GetString("BTC_RPC_USER")
	cfg.BTC.RPCPassword = viper.GetString("BTC_RPC_PASSWORD")
	cfg.BTC.SyncInterval = viper.GetInt("BTC_SYNC_INTERVAL")

	// Solana 配置
	cfg.SOL.RPCURL = viper.GetString("SOL_RPC_URL")
	cfg.SOL.SyncInterval = viper.GetInt("SOL_SYNC_INTERVAL")

	// 价格 API 配置
	cfg.Price.APIURL = viper.GetString("PRICE_API_URL")
	cfg.Price.SyncInterval = viper.GetInt("PRICE_SYNC_INTERVAL")

	// 服务端口配置
	cfg.Server.QueryAPIPort = viper.GetInt("QUERY_API_PORT")
	cfg.Server.SearchAPIPort = viper.GetInt("SEARCH_API_PORT")
	cfg.Server.PriceAPIPort = viper.GetInt("PRICE_API_PORT")

	// 日志配置
	cfg.Log.Level = viper.GetString("LOG_LEVEL")
	cfg.Log.Format = viper.GetString("LOG_FORMAT")

	// 代理配置（用于日志显示）
	cfg.Proxy.HTTP = viper.GetString("HTTP_PROXY")
	cfg.Proxy.HTTPS = viper.GetString("HTTPS_PROXY")

	return cfg
}

// ============================================================
// setDefaults 函数：设置所有配置项的默认值
// ============================================================
// 当配置文件和环境变量都没有设置时，使用这些默认值
func setDefaults() {
	// ---- 应用配置默认值 ----
	viper.SetDefault("APP_NAME", "blockexplore")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_DEBUG", true)

	// ---- 数据库配置默认值 ----
	// Docker 内部使用 "postgres" 作为主机名（Docker 服务名）
	viper.SetDefault("DB_HOST", "postgres")
	viper.SetDefault("DB_PORT", 5432) // PostgreSQL 默认端口
	viper.SetDefault("DB_USER", "blockexplore")
	viper.SetDefault("DB_PASSWORD", "blockexplore123")
	viper.SetDefault("DB_NAME", "blockexplore")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 100) // 最大连接数
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)  // 空闲连接数

	// ---- Redis 配置默认值 ----
	// Docker 内部使用 "redis" 作为主机名
	viper.SetDefault("REDIS_HOST", "redis")
	viper.SetDefault("REDIS_PORT", 6379) // Redis 默认端口
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)       // 使用 0 号数据库
	viper.SetDefault("REDIS_POOL_SIZE", 100)

	// ---- Kafka 配置默认值 ----
	viper.SetDefault("KAFKA_BROKERS", []string{"kafka:9092"})
	viper.SetDefault("KAFKA_ETH_TOPIC", "block.raw.eth")
	viper.SetDefault("KAFKA_BTC_TOPIC", "block.raw.btc")
	viper.SetDefault("KAFKA_SOL_TOPIC", "block.raw.sol")
	viper.SetDefault("KAFKA_CONSUMER_GROUP", "block-processor-group")

	// ---- 各链同步间隔默认值（秒）----
	// ETH: 约 12 秒一个区块
	// BTC: 约 10 分钟一个区块
	// SOL: 约 0.4 秒一个区块，但公开 RPC 限流，5 秒比较安全
	viper.SetDefault("ETH_SYNC_INTERVAL", 12)
	viper.SetDefault("BTC_SYNC_INTERVAL", 600)
	viper.SetDefault("SOL_SYNC_INTERVAL", 5)

	// ---- 价格 API 默认值 ----
	// CoinGecko 免费 API 限流严格，120 秒比较安全
	viper.SetDefault("PRICE_API_URL", "https://api.coingecko.com/api/v3")
	viper.SetDefault("PRICE_SYNC_INTERVAL", 120)

	// ---- 服务端口默认值 ----
	viper.SetDefault("QUERY_API_PORT", 8080)
	viper.SetDefault("SEARCH_API_PORT", 8081)
	viper.SetDefault("PRICE_API_PORT", 8082)

	// ---- 日志配置默认值 ----
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
}
```

> **注意 `RTCTopic` 这个字段名**：它实际存的是 BTC 的 topic，字段名是历史遗留（RTC ≠ BTC）。代码里全项目都用 `cfg.Kafka.RTCTopic` 访问 BTC topic，保持一致即可。

### 5.3 编译检查

```bash
go build ./internal/config/
```

无输出 = 编译成功。

<details>
<summary>报 "no required module provides package"？</summary>

运行 `go mod tidy` 自动补全依赖。

</details>

---

## 第 6 章：日志工具 —— logger 包

```bash
touch pkg/logger/logger.go
```

写入 `pkg/logger/logger.go`：

```go
// Package logger 封装 zap 日志库，提供统一的日志接口
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 全局日志实例
var log *zap.Logger

// Init 初始化日志系统
// level: 日志级别（debug/info/warn/error）
// format: 输出格式（json/console）
func Init(level, format string) {
	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 配置编码器（决定日志输出格式）
	var encoderConfig zapcore.EncoderConfig
	if format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// 时间字段名改为 timestamp，使用 ISO8601 格式
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 创建编码器实例
	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建日志核心：编码器 + 输出目标（stdout）+ 级别
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel)

	// 创建 Logger，附加调用者信息（显示日志来自哪个文件哪一行）
	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
}

// Get 获取日志实例（未初始化时返回生产模式默认实例）
func Get() *zap.Logger {
	if log == nil {
		log, _ = zap.NewProduction()
	}
	return log
}

// 以下是便捷函数，避免每次都写 logger.Get().Info(...)

func Debug(msg string, fields ...zap.Field) { Get().Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { Get().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { Get().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { Get().Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { Get().Fatal(msg, fields...) }
```

编译检查：

```bash
go build ./pkg/logger/
```

---

## 第 7 章：Redis 缓存工具 —— cache 包

```bash
touch pkg/cache/redis.go
```

写入 `pkg/cache/redis.go`（注意：这里是**结构体封装**的 `RedisClient`，不是全局函数）：

```go
// Package cache 封装 Redis 缓存操作
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blockexplore/internal/config"
	"blockexplore/pkg/logger"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisClient Redis 客户端封装
type RedisClient struct {
	client *redis.Client // Redis 客户端实例
}

// 全局 Redis 客户端实例
var rdb *RedisClient

// Init 初始化 Redis 连接
// 传入 Redis 配置，创建连接池
func Init(cfg config.RedisConfig) {
	rdb = &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr(),     // Redis 地址: host:port
			Password: cfg.Password,   // 密码（无密码留空）
			DB:       cfg.DB,         // 数据库编号
			PoolSize: cfg.PoolSize,   // 连接池大小
		}),
	}

	// 测试连接是否成功
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis 连接失败", zap.Error(err))
	} else {
		logger.Info("Redis 连接成功", zap.String("addr", cfg.Addr()))
	}
}

// GetClient 获取 Redis 客户端实例
func GetClient() *RedisClient {
	return rdb
}

// Set 设置缓存（带过期时间）
// key: 缓存键
// value: 缓存值（会被 JSON 序列化）
// expiration: 过期时间
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 将值序列化为 JSON 字节
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化缓存值失败: %w", err)
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
// key: 缓存键
// dest: 目标对象（会被 JSON 反序列化填充）
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return err // redis.Nil 表示键不存在
	}
	return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// SetNX 仅在键不存在时设置（用于分布式锁等场景）
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("序列化缓存值失败: %w", err)
	}
	return r.client.SetNX(ctx, key, data, expiration).Result()
}

// Incr 原子递增（用于计数器、限流等场景）
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire 设置键的过期时间
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
```

编译检查：

```bash
go build ./pkg/cache/
```

---

## 第 8 章：错误码工具 —— errcode 包

```bash
touch pkg/errcode/errcode.go
```

写入 `pkg/errcode/errcode.go`：

```go
// Package errcode 定义统一的错误码和错误响应格式
// 所有 API 返回都遵循统一格式: { code, message, data, request_id }
package errcode

// 错误码定义
// 约定: 200=成功, 4xx=客户端错误, 5xx=服务端错误
const (
	CodeSuccess       = 200 // 成功
	CodeBadRequest    = 400 // 请求参数错误
	CodeNotFound      = 404 // 资源不存在
	CodeInternalError = 500 // 服务器内部错误
	CodeDBError       = 501 // 数据库错误
	CodeCacheError    = 502 // 缓存错误
	CodeRPCError      = 503 // 区块链 RPC 调用错误
	CodeKafkaError    = 504 // 消息队列错误
)

// 错误消息映射
var codeMessages = map[int]string{
	CodeSuccess:       "success",
	CodeBadRequest:    "请求参数错误",
	CodeNotFound:      "资源不存在",
	CodeInternalError: "服务器内部错误",
	CodeDBError:       "数据库错误",
	CodeCacheError:    "缓存错误",
	CodeRPCError:      "区块链 RPC 调用错误",
	CodeKafkaError:    "消息队列错误",
}

// GetMsg 根据错误码获取对应的错误消息
func GetMsg(code int) string {
	msg, ok := codeMessages[code]
	if !ok {
		return "未知错误"
	}
	return msg
}

// Response 统一 API 响应结构体
type Response struct {
	Code      int         `json:"code"`       // 错误码
	Message   string      `json:"message"`    // 错误消息
	Data      interface{} `json:"data"`       // 响应数据
	RequestID string      `json:"request_id"` // 请求 ID（用于链路追踪）
}

// Success 构建成功响应
func Success(data interface{}, requestID string) *Response {
	return &Response{
		Code:      CodeSuccess,
		Message:   GetMsg(CodeSuccess),
		Data:      data,
		RequestID: requestID,
	}
}

// Error 构建错误响应
func Error(code int, requestID string) *Response {
	return &Response{
		Code:      code,
		Message:   GetMsg(code),
		Data:      nil,
		RequestID: requestID,
	}
}

// ErrorWithMsg 构建带自定义消息的错误响应
func ErrorWithMsg(code int, msg string, requestID string) *Response {
	return &Response{
		Code:      code,
		Message:   msg,
		Data:      nil,
		RequestID: requestID,
	}
}
```

编译检查：

```bash
go build ./pkg/errcode/
```

---
## 第 9 章：数据模型层 —— model 包

### 9.1 区块模型

```bash
touch internal/model/block.go
```

写入 `internal/model/block.go`：

```go
// ============================================================
// Package model 定义数据模型，使用 GORM 映射数据库表结构
// ============================================================
// 该包定义了所有数据库表对应的结构体（Model）。
//
// GORM 是 Go 语言的 ORM（对象关系映射）库，可以将结构体映射到数据库表。
// 类似于 Java 的 Hibernate 或 Python 的 SQLAlchemy。
//
// GORM 标签说明：
//   - gorm:"primaryKey"：标记为主键
//   - gorm:"autoIncrement"：自增
//   - gorm:"type:varchar(10)"：指定数据库字段类型
//   - gorm:"not null"：不允许为空
//   - gorm:"default:0"：默认值
//   - gorm:"autoCreateTime"：创建时自动填充时间
//   - json:"field_name"：JSON 序列化时的字段名
//
// Go 语言基础知识:
//   - struct：结构体，类似于 Java 的 class
//   - 反引号 `：用于定义标签（tag），标签是键值对，用于给字段附加元信息
//   - *int64：指针类型，指针可以为 nil（表示空值），普通类型不能为 nil
//   - time.Time：Go 标准库的时间类型
// ============================================================
package model

import "time" // 导入时间包

// ============================================================
// Block 区块表模型
// ============================================================
// 对应数据库中的 blocks 表，存储各链的区块信息
// 每个区块包含：链标识、区块高度、区块哈希、父区块哈希、出块时间等
type Block struct {
	// 主键 ID，自增，GORM 会自动管理
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth（以太坊）/ btc（比特币）/ sol（Solana）
	// varchar(10) 表示最大长度 10 的字符串
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 区块高度（区块编号），从 0 开始递增
	// 比特币创世区块高度为 0，以太坊创世区块高度为 0
	BlockNumber int64 `gorm:"not null" json:"block_number"`

	// 区块哈希，唯一标识一个区块
	// 以太坊哈希 66 字符（0x + 64 十六进制字符）
	// 比特币哈希 64 字符（纯十六进制）
	BlockHash string `gorm:"type:varchar(128);not null" json:"block_hash"`

	// 父区块哈希，用于链接区块形成链
	// 每个区块都包含前一个区块的哈希，形成区块链
	ParentHash string `gorm:"type:varchar(128)" json:"parent_hash"`

	// 出块时间，Unix 时间戳（从 1970-01-01 00:00:00 UTC 开始的秒数）
	Timestamp int64 `gorm:"not null" json:"timestamp"`

	// 区块内的交易数量
	TxCount int `gorm:"default:0" json:"tx_count"`

	// 已消耗 Gas（仅以太坊/Solana 使用）
	// Gas 是以太坊的计费单位，类似于手机话费
	GasUsed string `gorm:"type:text" json:"gas_used"`

	// Gas 上限（仅以太坊使用）
	// 每个区块有 Gas 上限，限制区块内交易的总 Gas 消耗
	GasLimit string `gorm:"type:text" json:"gas_limit"`

	// 区块大小（字节，仅比特币使用）
	SizeBytes int `json:"size_bytes"`

	// 难度值（仅比特币使用）
	// 比特币的挖矿难度，越高越难挖到区块
	Difficulty string `gorm:"type:text" json:"difficulty"`

	// 槽位号（仅 Solana 使用）
	// *int64 是指针类型，可以为 nil（表示非 Solana 链时为空）
	Slot *int64 `json:"slot"`

	// 记录创建时间，GORM 会在插入时自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 方法指定 GORM 使用的表名
// 如果不实现这个方法，GORM 会默认使用结构体名的复数形式（blocks）
// 这里显式指定表名为 "blocks"
func (Block) TableName() string {
	return "blocks"
}
```

### 9.2 交易模型

```bash
touch internal/model/transaction.go
```

写入 `internal/model/transaction.go`：

```go
// ============================================================
// Package model 定义数据模型，使用 GORM 映射数据库表结构
// ============================================================
// 该文件定义了交易表的数据模型。
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *int64：指针类型，可以为 nil，表示可选字段
//   - gorm:"..."：GORM 标签，定义数据库字段属性
//   - json:"..."：JSON 标签，定义 JSON 序列化时的字段名
//   - index：创建索引，加快查询速度
// ============================================================
package model

import "time"

// ============================================================
// Transaction 交易表模型
// ============================================================
// 对应数据库中的 transactions 表，存储各链的交易信息
// 每笔交易包含：链标识、交易哈希、发送方、接收方、金额等
type Transaction struct {
	// 主键 ID，自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth/btc/sol
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 交易哈希，唯一标识一笔交易
	// 以太坊交易哈希 66 字符（0x + 64 十六进制字符）
	// 比特币交易哈希 64 字符（纯十六进制）
	// Solana 交易签名 87-88 字符（Base58 编码）
	TxHash string `gorm:"type:varchar(128);not null" json:"tx_hash"`

	// 所在区块高度
	BlockNumber int64 `gorm:"not null" json:"block_number"`

	// 关联的区块表 ID（外键）
	// 通过这个字段可以关联查询区块信息
	BlockID int64 `json:"block_id"`

	// 发送方地址
	// index 标签会在这个字段上创建数据库索引，加快按地址查询的速度
	FromAddr string `gorm:"type:varchar(128);index" json:"from_addr"`

	// 接收方地址
	ToAddr string `gorm:"type:varchar(128);index" json:"to_addr"`

	// 转账金额
	// numeric(78,18) 表示最多 78 位数字，其中小数部分 18 位
	// 使用字符串存储是因为 Go 的 float64 精度不够，大金额会丢失精度
	Value string `gorm:"type:numeric(78,18)" json:"value"`

	// Gas 价格（仅以太坊使用）
	// Gas 价格越高，矿工越优先打包你的交易
	GasPrice string `gorm:"type:text" json:"gas_price"`

	// 实际消耗 Gas（仅以太坊使用）
	// 交易执行后实际消耗的 Gas 数量
	GasUsed string `gorm:"type:text" json:"gas_used"`

	// Gas 上限（仅以太坊使用）
	// 用户愿意为这笔交易支付的最大 Gas 数量
	GasLimit string `gorm:"type:text" json:"gas_limit"`

	// 交易序号（仅以太坊使用）
	// 每个账户的交易都有一个递增的序号，防止重放攻击
	Nonce *int64 `json:"nonce"`

	// 调用数据（仅以太坊使用）
	// 如果是合约调用，这里存储调用参数（calldata）
	InputData string `gorm:"type:text" json:"input_data"`

	// 交易状态：1=成功，0=失败
	Status int16 `gorm:"default:1" json:"status"`

	// 交易时间，Unix 时间戳
	Timestamp int64 `gorm:"not null" json:"timestamp"`

	// 记录创建时间，GORM 自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 方法指定 GORM 使用的表名
func (Transaction) TableName() string {
	return "transactions"
}
```

> **关键点：`Value` 用 `numeric(78,18)` 而不是 `text`**。这是真实项目使用的类型，支持高精度大数。集成测试发现：插入 `Value=""` 会报 `invalid input syntax for type numeric` 错误，所以代码里所有交易写入前都会给 `Value` 赋值（如 `"0"`）。

### 9.3 地址模型

```bash
touch internal/model/address.go
```

写入 `internal/model/address.go`：

```go
// ============================================================
// Address 地址表模型
// ============================================================
// 对应数据库中的 addresses 表，记录地址的交易统计信息。
//
// 地址是区块链上的账户标识：
//   - 以太坊地址：42 字符，0x 开头，例如 0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18
//   - 比特币地址：以 1、3 或 bc1 开头
//   - Solana 地址：32-44 字符的 Base58 编码
//
// Go 语言基础知识:
//   - struct：结构体，类似于 Java 的 class
//   - gorm:"autoCreateTime"：插入记录时自动填充创建时间
//   - gorm:"autoUpdateTime"：更新记录时自动填充更新时间
// ============================================================
package model

import "time"

// ============================================================
// Address 地址表模型
// ============================================================
// 记录每个地址的余额、交易次数、首次/最近交易时间等统计信息
type Address struct {
	// 主键 ID，自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth/btc/sol
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 区块链地址
	Address string `gorm:"type:varchar(128);not null" json:"address"`

	// 当前余额
	// numeric(78,18) 支持非常大的数字，小数部分 18 位
	// 以太坊的最小单位是 Wei，1 ETH = 10^18 Wei
	Balance string `gorm:"type:numeric(78,18)" json:"balance"`

	// 交易总数（该地址参与的所有交易）
	TxCount int64 `gorm:"default:0" json:"tx_count"`

	// 首次交易时间（Unix 时间戳）
	FirstSeenAt int64 `json:"first_seen_at"`

	// 最近交易时间（Unix 时间戳）
	LastSeenAt int64 `json:"last_seen_at"`

	// 记录创建时间，GORM 插入时自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 记录更新时间，GORM 更新时自动填充
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 方法指定 GORM 使用的表名
func (Address) TableName() string {
	return "addresses"
}
```

> **这个文件绝对不能漏！** `search_repo.go` 的搜索逻辑会查 `addresses` 表，缺了这个 model 编译会失败。

### 9.4 价格模型

```bash
touch internal/model/price.go
```

写入 `internal/model/price.go`：

```go
// ============================================================
// PriceHistory 价格历史表模型
// ============================================================
// 对应数据库中的 price_history 表，记录各链原生代币的历史价格。
//
// 用途：
//   - 绘制价格曲线图
//   - 查询历史价格
//   - 价格趋势分析
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据模型
//   - gorm 标签：定义数据库字段属性
//   - json 标签：定义 JSON 序列化时的字段名
// ============================================================
package model

import "time"

// ============================================================
// PriceHistory 价格历史表模型
// ============================================================
// 记录 ETH、BTC、SOL 的历史价格数据
// 数据来源：CoinGecko API（免费的加密货币价格 API）
type PriceHistory struct {
	// 主键 ID，自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth/btc/sol
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 代币符号：ETH/BTC/SOL
	Symbol string `gorm:"type:varchar(10);not null" json:"symbol"`

	// 美元价格
	// 使用 text 类型存储，因为价格可能有小数，精度要求高
	PriceUSD string `gorm:"type:text" json:"price_usd"`

	// 价格时间，Unix 时间戳
	// 记录这个价格是什么时候获取的
	Timestamp int64 `gorm:"not null" json:"timestamp"`

	// 记录创建时间，GORM 自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 方法指定 GORM 使用的表名
func (PriceHistory) TableName() string {
	return "price_history"
}
```

编译检查：

```bash
go build ./internal/model/
```

---

## 第 10 章：数据访问层 —— repository 包

### 10.1 区块 Repository

```bash
touch internal/repository/block_repo.go
```

写入 `internal/repository/block_repo.go`（142 行，与项目实际代码一致）：

```go
// ============================================================
// Package repository 提供数据访问层，封装数据库操作
// ============================================================
// 该包是数据访问层（DAO），负责与数据库交互。
//
// 分层架构：
//   - Handler（控制器层）：接收 HTTP 请求，调用 Service
//   - Service（业务逻辑层）：处理业务逻辑，调用 Repository
//   - Repository（数据访问层）：执行数据库查询，返回结果
//
// 为什么要分层？
//   - 解耦：业务逻辑和数据访问分离，方便维护
//   - 可测试：可以 Mock Repository 层，方便单元测试
//   - 复用：同一个 Repository 方法可以被多个 Service 调用
//
// Go 语言基础知识:
//   - package：包，Go 的模块化机制
//   - struct：结构体，用于定义数据结构
//   - *gorm.DB：GORM 数据库连接的指针
//   - error：Go 的错误类型，函数通过返回 error 来报告错误
//   - .Error：GORM 的错误属性，操作失败时会设置
//   - append：向切片追加元素
// ============================================================
package repository

import (
	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// BlockRepo 区块数据访问层
// ============================================================
// 封装了区块相关的数据库操作
type BlockRepo struct {
	db *gorm.DB // 数据库连接实例
}

// ============================================================
// NewBlockRepo 创建区块数据访问层实例
// ============================================================
// 参数 db：GORM 数据库连接
// 返回值：*BlockRepo 指针
func NewBlockRepo(db *gorm.DB) *BlockRepo {
	return &BlockRepo{db: db}
}

// ============================================================
// Create 方法：批量创建区块记录
// ============================================================
// 使用 GORM 的 CreateInBatches 批量插入，提高写入性能
// 参数 blocks：区块切片
// 参数 100：每批插入 100 条记录
func (r *BlockRepo) Create(blocks []model.Block) error {
	if len(blocks) == 0 {
		return nil // 空切片直接返回，不执行数据库操作
	}
	return r.db.CreateInBatches(blocks, 100).Error
}

// ============================================================
// CreateSingle 方法：创建单个区块记录
// ============================================================
// 参数 block：区块指针
// GORM 会自动填充 ID、CreatedAt 等字段
func (r *BlockRepo) CreateSingle(block *model.Block) error {
	return r.db.Create(block).Error
}

// ============================================================
// GetByChainAndNumber 方法：根据链标识和区块高度查询区块
// ============================================================
// 参数 chain：链标识（eth/btc/sol）
// 参数 blockNumber：区块高度
// 返回值：区块指针和错误信息
func (r *BlockRepo) GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error) {
	var block model.Block
	// Where 添加查询条件，First 查询第一条记录
	// ? 是参数占位符，防止 SQL 注入
	err := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber).First(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// ============================================================
// GetLatest 方法：获取指定链的最新区块
// ============================================================
// 按区块高度降序排列，取第一条
func (r *BlockRepo) GetLatest(chain string) (*model.Block, error) {
	var block model.Block
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC"). // 降序排列
		First(&block).Error        // 取第一条
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// ============================================================
// GetList 方法：获取区块列表（分页）
// ============================================================
// 参数 chain：链标识
// 参数 page：页码（从 1 开始）
// 参数 pageSize：每页数量
// 返回值：区块切片、总数、错误信息
func (r *BlockRepo) GetList(chain string, page, pageSize int) ([]model.Block, int64, error) {
	var blocks []model.Block
	var total int64

	// 查询总数
	// Model 指定模型，Count 统计总数
	r.db.Model(&model.Block{}).Where("chain = ?", chain).Count(&total)

	// 分页查询
	// offset = (page - 1) * pageSize，跳过前面的记录
	offset := (page - 1) * pageSize
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC"). // 按区块高度降序，最新的在前
		Offset(offset).             // 跳过 offset 条记录
		Limit(pageSize).            // 最多返回 pageSize 条记录
		Find(&blocks).Error         // 查询结果填充到 blocks

	return blocks, total, err
}

// ============================================================
// GetLatestN 方法：获取指定链的最新 N 个区块
// ============================================================
// 参数 chain：链标识
// 参数 n：数量
func (r *BlockRepo) GetLatestN(chain string, n int) ([]model.Block, error) {
	var blocks []model.Block
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC").
		Limit(n).
		Find(&blocks).Error
	return blocks, err
}
```

### 10.2 交易 Repository

```bash
touch internal/repository/tx_repo.go
```

写入 `internal/repository/tx_repo.go`（注意方法名 `GetByBlockNumber` 不是 `GetByBlockID`）：

```go
// ============================================================
// TxRepo 交易数据访问层
// ============================================================
// 封装了交易相关的数据库操作。
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *gorm.DB：GORM 数据库连接的指针
//   - error：Go 的错误类型
//   - OR 条件：GORM 使用 OR() 方法或直接在 Where 中写 SQL
//   - 分页：Offset 跳过记录，Limit 限制数量
// ============================================================
package repository

import (
	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// TxRepo 交易数据访问层
// ============================================================
type TxRepo struct {
	db *gorm.DB
}

// ============================================================
// NewTxRepo 创建交易数据访问层实例
// ============================================================
func NewTxRepo(db *gorm.DB) *TxRepo {
	return &TxRepo{db: db}
}

// ============================================================
// Create 方法：批量创建交易记录
// ============================================================
// 使用 CreateInBatches 批量插入，每批 100 条
func (r *TxRepo) Create(txs []model.Transaction) error {
	if len(txs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(txs, 100).Error
}

// ============================================================
// CreateSingle 方法：创建单个交易记录
// ============================================================
func (r *TxRepo) CreateSingle(tx *model.Transaction) error {
	return r.db.Create(tx).Error
}

// ============================================================
// GetByHash 方法：根据链标识和交易哈希查询交易
// ============================================================
// 参数 chain：链标识（eth/btc/sol）
// 参数 txHash：交易哈希
func (r *TxRepo) GetByHash(chain, txHash string) (*model.Transaction, error) {
	var tx model.Transaction
	err := r.db.Where("chain = ? AND tx_hash = ?", chain, txHash).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// ============================================================
// GetByBlockNumber 方法：获取指定区块内的所有交易（分页）
// ============================================================
// 参数 chain：链标识
// 参数 blockNumber：区块高度
// 参数 page：页码
// 参数 pageSize：每页数量
func (r *TxRepo) GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	// 构建查询条件
	query := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber)
	// 统计总数
	query.Model(&model.Transaction{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("id ASC"). // 按 ID 升序
					Offset(offset).
					Limit(pageSize).
					Find(&txs).Error

	return txs, total, err
}

// ============================================================
// GetByAddress 方法：获取指定地址的交易记录（分页）
// ============================================================
// 同时查询 from_addr 和 to_addr，即该地址作为发送方或接收方的交易
// 使用 OR 条件：from_addr = address OR to_addr = address
func (r *TxRepo) GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	// SQL: WHERE chain = ? AND (from_addr = ? OR to_addr = ?)
	query := r.db.Where("chain = ? AND (from_addr = ? OR to_addr = ?)", chain, address, address)
	query.Model(&model.Transaction{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("timestamp DESC"). // 按时间降序，最新的在前
					Offset(offset).
					Limit(pageSize).
					Find(&txs).Error

	return txs, total, err
}

// ============================================================
// GetLatestN 方法：获取指定链的最新 N 条交易
// ============================================================
func (r *TxRepo) GetLatestN(chain string, n int) ([]model.Transaction, error) {
	var txs []model.Transaction
	err := r.db.Where("chain = ?", chain).
		Order("timestamp DESC").
		Limit(n).
		Find(&txs).Error
	return txs, err
}
```

### 10.3 搜索 Repository

```bash
touch internal/repository/search_repo.go
```

写入 `internal/repository/search_repo.go`（注意：是**单一 `Search(keyword)` 方法**，不是两个分开的方法）：

```go
// ============================================================
// SearchRepo 搜索数据访问层
// ============================================================
// 提供统一搜索功能，根据输入自动识别类型（区块/交易/地址）。
//
// 搜索逻辑：
//   - 纯数字 → 区块高度
//   - 0x 开头 + 64 字符 → 以太坊交易哈希
//   - 0x 开头 + 40 字符 → 以太坊地址
//   - 1/3/bc1 开头 → 比特币地址
//   - Base58 编码 → Solana 交易签名或地址
//
// Go 语言基础知识:
//   - regexp.MustCompile：编译正则表达式，Must 表示如果编译失败会 panic
//   - strconv.ParseInt：字符串转整数
//   - interface{}：空接口，可以持有任意类型的值
//   - var isNumberRegex = regexp.MustCompile(...)：包级变量，程序启动时编译一次
//   - keyword[:2]：切片操作，取前 2 个字符
//   - keyword[0]：取第一个字符
// ============================================================
package repository

import (
	"regexp"  // 正则表达式
	"strconv" // 字符串转换

	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// SearchRepo 搜索数据访问层
// ============================================================
type SearchRepo struct {
	db *gorm.DB
}

// ============================================================
// NewSearchRepo 创建搜索数据访问层实例
// ============================================================
func NewSearchRepo(db *gorm.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

// ============================================================
// SearchResult 搜索结果结构体
// ============================================================
type SearchResult struct {
	Type string      `json:"type"` // 结果类型: block/transaction/address
	Data interface{} `json:"data"` // 结果数据（可以是任意类型）
}

// ============================================================
// 匹配规则说明
// ============================================================
// 根据输入内容的格式判断搜索类型：
//   - 42 字符，0x 开头 → 以太坊地址
//   - 66 字符，0x 开头 → 以太坊交易哈希
//   - 64 字符，非 0x 开头 → 比特币交易哈希
//   - 87-88 字符，Base58 → Solana 签名
//   - 1/3/bc1 开头 → 比特币地址
//   - 纯数字 → 区块高度

// isNumberRegex 判断是否是纯数字（区块高度）
// ^ 表示字符串开头，\d+ 表示一个或多个数字，$ 表示字符串结尾
var isNumberRegex = regexp.MustCompile(`^\d+$`)

// ============================================================
// Search 方法：统一搜索入口
// ============================================================
// 参数 keyword：搜索关键词
// 返回值：搜索结果（包含类型和数据），未找到返回 nil
func (r *SearchRepo) Search(keyword string) (*SearchResult, error) {
	// ============================================================
	// 第 1 步：判断是否是区块高度（纯数字）
	// ============================================================
	if isNumberRegex.MatchString(keyword) {
		// strconv.ParseInt 将字符串解析为 int64
		// 参数：字符串、进制（10=十进制）、位数（64 位）
		blockNumber, err := strconv.ParseInt(keyword, 10, 64)
		if err == nil {
			// 尝试在三条链中查找
			for _, chain := range []string{"eth", "btc", "sol"} {
				var block model.Block
				if err := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber).First(&block).Error; err == nil {
					return &SearchResult{Type: "block", Data: block}, nil
				}
			}
		}
	}

	// ============================================================
	// 第 2 步：判断是否是 0x 开头（以太坊交易哈希或地址）
	// ============================================================
	// 以太坊地址：42 字符（0x + 40 十六进制字符）
	// 以太坊交易哈希：66 字符（0x + 64 十六进制字符）
	if len(keyword) == 66 && keyword[:2] == "0x" {
		// 先查交易
		var tx model.Transaction
		if err := r.db.Where("chain = 'eth' AND tx_hash = ?", keyword).First(&tx).Error; err == nil {
			return &SearchResult{Type: "transaction", Data: tx}, nil
		}
		// 再查地址
		var addr model.Address
		if err := r.db.Where("chain = 'eth' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
	}

	// 0x 开头但不是 66 字符，可能是 42 字符的地址
	if len(keyword) == 42 && keyword[:2] == "0x" {
		var addr model.Address
		if err := r.db.Where("chain = 'eth' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
	}

	// ============================================================
	// 第 3 步：判断是否是比特币地址（1/3/bc1 开头）
	// ============================================================
	// 比特币地址格式：
	//   - P2PKH：以 1 开头，例如 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
	//   - P2SH：以 3 开头，例如 3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy
	//   - Bech32：以 bc1 开头，例如 bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq
	if len(keyword) > 0 && (keyword[0] == '1' || keyword[0] == '3' || keyword[:3] == "bc1") {
		var addr model.Address
		if err := r.db.Where("chain = 'btc' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
		// 也可能是比特币交易哈希
		var tx model.Transaction
		if err := r.db.Where("chain = 'btc' AND tx_hash = ?", keyword).First(&tx).Error; err == nil {
			return &SearchResult{Type: "transaction", Data: tx}, nil
		}
	}

	// ============================================================
	// 第 4 步：判断是否是 Solana 签名或地址
	// ============================================================
	// Solana 交易签名：87-88 字符的 Base58 编码
	// Solana 地址：32-44 字符的 Base58 编码
	if len(keyword) >= 32 && len(keyword) <= 88 {
		// 先查交易
		var tx model.Transaction
		if err := r.db.Where("chain = 'sol' AND tx_hash = ?", keyword).First(&tx).Error; err == nil {
			return &SearchResult{Type: "transaction", Data: tx}, nil
		}
		// 再查地址
		var addr model.Address
		if err := r.db.Where("chain = 'sol' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
	}

	return nil, nil // 未找到匹配结果
}
```

### 10.4 价格 Repository

```bash
touch internal/repository/price_repo.go
```

写入 `internal/repository/price_repo.go`（注意方法名 `GetLatestPrice` 和 `GetPriceHistory`）：

```go
// ============================================================
// PriceRepo 价格数据访问层
// ============================================================
// 封装了价格历史相关的数据库操作。
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *gorm.DB：GORM 数据库连接的指针
//   - error：Go 的错误类型
//   - 条件查询：Where 添加条件，支持链式调用
//   - 时间范围查询：timestamp >= ? AND timestamp <= ?
// ============================================================
package repository

import (
	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// PriceRepo 价格数据访问层
// ============================================================
type PriceRepo struct {
	db *gorm.DB
}

// ============================================================
// NewPriceRepo 创建价格数据访问层实例
// ============================================================
func NewPriceRepo(db *gorm.DB) *PriceRepo {
	return &PriceRepo{db: db}
}

// ============================================================
// Create 方法：保存价格记录
// ============================================================
// 参数 price：价格历史记录指针
func (r *PriceRepo) Create(price *model.PriceHistory) error {
	return r.db.Create(price).Error
}

// ============================================================
// CreateBatch 方法：批量保存价格记录
// ============================================================
func (r *PriceRepo) CreateBatch(prices []model.PriceHistory) error {
	if len(prices) == 0 {
		return nil
	}
	return r.db.CreateInBatches(prices, 100).Error
}

// ============================================================
// GetLatestPrice 方法：获取指定链的最新价格
// ============================================================
// 按时间戳降序排列，取第一条
func (r *PriceRepo) GetLatestPrice(chain string) (*model.PriceHistory, error) {
	var price model.PriceHistory
	err := r.db.Where("chain = ?", chain).
		Order("timestamp DESC").
		First(&price).Error
	if err != nil {
		return nil, err
	}
	return &price, nil
}

// ============================================================
// GetPriceHistory 方法：获取价格历史（用于绘制价格曲线）
// ============================================================
// 参数 chain：链标识
// 参数 startTime：开始时间（Unix 时间戳），0 表示不限制
// 参数 endTime：结束时间（Unix 时间戳），0 表示不限制
// 参数 limit：最大返回条数
func (r *PriceRepo) GetPriceHistory(chain string, startTime, endTime int64, limit int) ([]model.PriceHistory, error) {
	var prices []model.PriceHistory
	query := r.db.Where("chain = ?", chain)

	// 时间范围条件
	if startTime > 0 {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("timestamp <= ?", endTime)
	}

	// 按时间升序排列，限制返回条数
	err := query.Order("timestamp ASC").
		Limit(limit).
		Find(&prices).Error

	return prices, err
}
```

编译检查：

```bash
go build ./internal/repository/
```

---

## 第 11 章：Kafka 消息队列 —— mq 包

Kafka 是项目的"消息中枢"。三个 Sync Worker 把区块数据写入 Kafka，Block Processor 从 Kafka 读出来处理。

```
eth-sync-worker ──→ Kafka Topic "block.raw.eth" ──→ block-processor
btc-sync-worker ──→ Kafka Topic "block.raw.btc" ──→ block-processor
sol-sync-worker ──→ Kafka Topic "block.raw.sol" ──→ block-processor
```

### 11.1 Kafka 生产者

```bash
touch internal/mq/producer.go
```

写入 `internal/mq/producer.go`（152 行，与项目实际代码一致）：

```go
// ============================================================
// Package mq 封装 Kafka 消息队列的生产者和消费者
// ============================================================
// 该包提供了 Kafka 消息队列的操作封装。
//
// Kafka 是什么？
//   - Apache Kafka 是一个分布式流处理平台
//   - 用于在不同服务之间传递消息
//   - 类似于一个"消息管道"，生产者往里写消息，消费者从里面读消息
//
// 本项目中 Kafka 的作用：
//   - 生产者（Sync Worker）：从区块链节点拉取区块数据，发送到 Kafka
//   - 消费者（Block Processor）：从 Kafka 读取区块数据，解析后写入数据库
//
// 为什么使用 Kafka？
//   - 解耦：Sync Worker 和 Block Processor 互不依赖
//   - 削峰：区块数据量大时，Kafka 可以缓冲消息，防止数据库压力过大
//   - 可靠：消息持久化，即使消费者挂了也不会丢失
//
// Go 语言基础知识:
//   - package：包，Go 的模块化机制
//   - struct：结构体，用于定义数据结构
//   - context.Context：上下文，用于控制超时和取消
//   - interface{}：空接口，可以持有任意类型的值
//   - json.Marshal：将结构体序列化为 JSON 字节
//   - fmt.Errorf：格式化创建错误，%w 包装原始错误
//   - defer：延迟执行，确保资源被正确释放
// ============================================================
package mq

import (
	"context"       // 上下文，用于超时控制
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串

	"blockexplore/internal/config" // 配置管理
	"blockexplore/pkg/logger"     // 日志

	"github.com/segmentio/kafka-go" // Kafka Go 客户端库
	"go.uber.org/zap"              // 日志库
)

// ============================================================
// Producer Kafka 生产者
// ============================================================
// 负责将消息发送到指定的 Kafka Topic
// Topic 是 Kafka 中消息的分类，类似于邮件的收件箱
type Producer struct {
	writer *kafka.Writer // Kafka 写入器，用于发送消息
	topic  string        // 目标 Topic 名称
}

// ============================================================
// BlockMessage 发送到 Kafka 的区块消息格式
// ============================================================
// Sync Worker 将区块数据封装为此格式发送到 Kafka
// Block Processor 从 Kafka 读取消息后，反序列化为这个格式
type BlockMessage struct {
	Chain       string      `json:"chain"`        // 链标识: eth/btc/sol
	BlockNumber int64       `json:"block_number"` // 区块高度
	Data        interface{} `json:"data"`          // 原始区块数据（使用空接口，可以存放任意类型）
}

// ============================================================
// NewProducer 创建 Kafka 生产者实例
// ============================================================
// 参数 brokers：Kafka Broker 地址列表，例如 ["kafka:9092"]
// 参数 topic：目标 Topic 名称，例如 "block.raw.eth"
func NewProducer(brokers []string, topic string) *Producer {
	// kafka.NewWriter 创建 Kafka 写入器
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...), // Broker 地址，... 展开切片
		Topic:        topic,                  // 目标 Topic
		Balancer:     &kafka.LeastBytes{},   // 负载均衡策略：选择字节数最少的分区
		RequiredAcks: kafka.RequireOne,       // 确认机制：至少一个 Broker 确认收到
		Async:        false,                  // 同步写入，确保消息不丢失（异步写入更快但可能丢消息）
	}

	logger.Info("Kafka 生产者已创建",
		zap.String("topic", topic),
		zap.Strings("brokers", brokers),
	)

	return &Producer{
		writer: writer,
		topic:  topic,
	}
}

// ============================================================
// Send 方法：发送消息到 Kafka
// ============================================================
// 参数 ctx：上下文，用于超时控制
// 参数 msg：要发送的区块消息
func (p *Producer) Send(ctx context.Context, msg BlockMessage) error {
	// 将消息序列化为 JSON 字节
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 构建 Kafka 消息
	// Key 用于决定消息发送到哪个分区（相同 Key 的消息会发送到同一分区）
	// 这里使用 "链标识-区块高度" 作为 Key
	kafkaMsg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s-%d", msg.Chain, msg.BlockNumber)),
		Value: data,
	}

	// 发送消息
	if err := p.writer.WriteMessages(ctx, kafkaMsg); err != nil {
		return fmt.Errorf("发送 Kafka 消息失败: %w", err)
	}

	logger.Debug("消息已发送到 Kafka",
		zap.String("topic", p.topic),
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
	)

	return nil
}

// ============================================================
// Close 方法：关闭生产者
// ============================================================
// 释放资源，关闭与 Kafka Broker 的连接
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// ============================================================
// 便捷创建函数
// ============================================================

// NewETHProducer 创建以太坊区块数据的 Kafka 生产者
func NewETHProducer(cfg config.KafkaConfig) *Producer {
	return NewProducer(cfg.Brokers, cfg.ETHTopic)
}

// NewBTCProducer 创建比特币区块数据的 Kafka 生产者
func NewBTCProducer(cfg config.KafkaConfig) *Producer {
	return NewProducer(cfg.Brokers, cfg.RTCTopic)
}

// NewSOLProducer 创建 Solana 区块数据的 Kafka 生产者
func NewSOLProducer(cfg config.KafkaConfig) *Producer {
	return NewProducer(cfg.Brokers, cfg.SOLTopic)
}
```

### 11.2 Kafka 消费者

```bash
touch internal/mq/consumer.go
```

写入 `internal/mq/consumer.go`（212 行，与项目实际代码一致）：

```go
// ============================================================
// Package mq 封装 Kafka 消息队列的生产者和消费者
// ============================================================
// 该文件实现了 Kafka 消费者。
//
// 消费者的工作流程：
//   1. 连接到 Kafka Broker
//   2. 订阅指定的 Topic
//   3. 循环读取消息
//   4. 将消息反序列化为 BlockMessage
//   5. 调用业务处理函数处理消息
//
// 消费者组（Consumer Group）：
//   - 同一个消费者组内的消费者共享消息
//   - 每条消息只会被组内的一个消费者处理
//   - 这样可以实现负载均衡和高可用
//
// Go 语言基础知识:
//   - goroutine：Go 的轻量级线程，用 go 关键字启动
//   - channel：Go 的通道，用于 goroutine 之间的通信
//   - select：多路复用，同时等待多个 channel 操作
//   - for { select { ... } }：Go 的经典事件循环模式
//   - ctx.Done()：返回一个 channel，当上下文被取消时会关闭
//   - func 类型：Go 中函数也是类型，可以作为参数传递
// ============================================================
package mq

import (
	"context"       // 上下文，用于控制 goroutine 的生命周期
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串

	"blockexplore/pkg/logger" // 日志

	"github.com/segmentio/kafka-go" // Kafka Go 客户端库
	"go.uber.org/zap"              // 日志库
)

// ============================================================
// Consumer Kafka 消费者
// ============================================================
// 负责从 Kafka Topic 消费消息并交给业务逻辑处理
type Consumer struct {
	reader *kafka.Reader // Kafka 读取器，用于读取消息
	topic  string        // 消费的 Topic 名称
}

// ============================================================
// MessageHandler 消息处理函数类型
// ============================================================
// 这是一个函数类型定义，类似于 Java 的接口
// 消费者收到消息后会调用此函数进行业务处理
// 函数参数是 BlockMessage，返回 error
type MessageHandler func(msg BlockMessage) error

// ============================================================
// NewConsumer 创建 Kafka 消费者实例
// ============================================================
// 参数 brokers：Kafka Broker 地址列表
// 参数 topic：消费的 Topic 名称
// 参数 group：消费者组名称（同一组内的消费者共享消息）
func NewConsumer(brokers []string, topic string, group string) *Consumer {
	// kafka.NewReader 创建 Kafka 读取器
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,       // Broker 地址列表
		Topic:    topic,         // 消费的 Topic
		GroupID:  group,         // 消费者组 ID
		MinBytes: 1,             // 最小拉取字节数（1 字节，尽快返回）
		MaxBytes: 10e6,          // 最大拉取字节数（10MB）
	})

	logger.Info("Kafka 消费者已创建",
		zap.String("topic", topic),
		zap.String("group", group),
		zap.Strings("brokers", brokers),
	)

	return &Consumer{
		reader: reader,
		topic:  topic,
	}
}

// ============================================================
// Consume 方法：开始消费消息
// ============================================================
// 参数 ctx：上下文（用于优雅关闭）
// 参数 handler：消息处理函数
// 此方法会阻塞，直到 ctx 被取消
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	logger.Info("开始消费 Kafka 消息", zap.String("topic", c.topic))

	// 无限循环，持续消费消息
	for {
		// select 语句用于多路复用，同时等待多个 channel 操作
		// 类似于 switch，但每个 case 都是 channel 操作
		select {
		case <-ctx.Done():
			// ctx.Done() 返回一个 channel
			// 当 ctx 被取消时，这个 channel 会被关闭
			// 从已关闭的 channel 读取会立即返回零值
			logger.Info("停止消费 Kafka 消息", zap.String("topic", c.topic))
			return nil
		default:
			// default 分支：如果没有其他 case 就绪，立即执行 default
			// 这样不会阻塞在 select 上，而是继续读取消息

			// 读取消息（阻塞直到有新消息或 ctx 被取消）
			kafkaMsg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				// 检查是否是上下文取消导致的错误
				if ctx.Err() != nil {
					return nil // 正常关闭，返回 nil
				}
				logger.Error("读取 Kafka 消息失败", zap.String("topic", c.topic), zap.Error(err))
				continue // 读取失败，跳过这条消息，继续读取下一条
			}

			// 反序列化消息
			// kafkaMsg.Value 是消息的原始字节，需要反序列化为 BlockMessage
			var msg BlockMessage
			if err := json.Unmarshal(kafkaMsg.Value, &msg); err != nil {
				logger.Error("解析 Kafka 消息失败",
					zap.String("topic", c.topic),
					zap.ByteString("value", kafkaMsg.Value), // 记录原始消息内容，方便调试
					zap.Error(err),
				)
				continue // 解析失败，跳过这条消息
			}

			// 调用业务处理函数
			// handler 是外部传入的处理函数，实现了具体的业务逻辑
			if err := handler(msg); err != nil {
				logger.Error("处理消息失败",
					zap.String("topic", c.topic),
					zap.String("chain", msg.Chain),
					zap.Int64("block_number", msg.BlockNumber),
					zap.Error(err),
				)
				// 处理失败可以选择重试或跳过，这里选择继续
				continue
			}

			logger.Debug("消息处理成功",
				zap.String("topic", c.topic),
				zap.String("chain", msg.Chain),
				zap.Int64("block_number", msg.BlockNumber),
			)
		}
	}
}

// ============================================================
// Close 方法：关闭消费者
// ============================================================
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// ============================================================
// NewBlockConsumer 创建区块数据消费者
// ============================================================
// 消费所有链的区块数据 Topic
func NewBlockConsumer(cfg struct {
	Brokers       []string
	ConsumerGroup string
}, topics ...string) []*Consumer {
	consumers := make([]*Consumer, 0, len(topics))
	for _, topic := range topics {
		consumer := NewConsumer(cfg.Brokers, topic, cfg.ConsumerGroup)
		consumers = append(consumers, consumer)
	}
	return consumers
}

// ============================================================
// ConsumeAll 并发消费多个 Topic
// ============================================================
// 参数 ctx：上下文
// 参数 consumers：消费者列表
// 参数 handler：统一的消息处理函数
// 使用 goroutine 并发消费，任意一个出错则返回错误
func ConsumeAll(ctx context.Context, consumers []*Consumer, handler MessageHandler) error {
	// errChan 是一个带缓冲的 channel，用于接收错误
	// 缓冲区大小为消费者数量，防止 goroutine 阻塞
	errChan := make(chan error, len(consumers))

	// 为每个消费者启动一个 goroutine
	for _, consumer := range consumers {
		// go func(c *Consumer) { ... }(consumer) 启动 goroutine
		// 注意：这里传入 consumer 参数，避免闭包捕获循环变量的问题
		go func(c *Consumer) {
			if err := c.Consume(ctx, handler); err != nil {
				errChan <- fmt.Errorf("消费者 %s 出错: %w", c.topic, err)
			}
		}(consumer)
	}

	// 等待第一个错误或上下文取消
	// select 会阻塞，直到某个 case 就绪
	select {
	case err := <-errChan:
		// 从 errChan 读取到错误
		return err
	case <-ctx.Done():
		// 上下文被取消（正常关闭）
		return nil
	}
}
```

编译检查：

```bash
go build ./internal/mq/
```

> **完整代码说明**：`consumer.go` 还包含 `NewBlockConsumer` 和 `ConsumeAll` 两个辅助函数。`NewBlockConsumer` 接收一个匿名结构体参数（含 `Brokers` 和 `ConsumerGroup` 字段）和可变 topic 列表，返回消费者切片。`ConsumeAll` 用 goroutine 并发消费多个 topic，任一出错即返回。

---

## 第 12 章：区块链客户端 —— client 包

客户端负责跟各条链的节点通信，拉取原始区块数据。三条链用三种不同的协议，但都封装成统一的 `(*model.Block, []model.Transaction, error)` 返回值。

> **重要说明**：本章代码很长（ETH 431 行、BTC 232 行、SOL 317 行）。**代码必须与项目仓库中的实际文件逐字一致**——本指南受篇幅限制只展示关键结构和函数签名，完整源码请直接从项目仓库的 `internal/client/` 目录复制对应文件。

### 12.1 以太坊客户端（JSON-RPC）

```bash
touch internal/client/eth_client.go
```

关键结构（完整 431 行请对照仓库 `internal/client/eth_client.go`）：

```go
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"blockexplore/internal/model"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

type EthClient struct {
	rpcURL     string
	httpClient *http.Client
}

func NewEthClient(rpcURL string) *EthClient {
	logProxyInfo("ETH", rpcURL)
	return &EthClient{
		rpcURL:     rpcURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// JSON-RPC 请求/响应结构体
type jsonRPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type jsonRPCResponse struct {
	JsonRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
	ID      int             `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC 错误 %d: %s", e.Code, e.Message)
}

// call 发送 JSON-RPC 请求（变参 params）
func (c *EthClient) call(method string, params ...interface{}) (json.RawMessage, error) {
	// 构建请求体 -> POST -> 解析响应 -> 检查 Error -> 返回 Result
	// 完整实现见仓库文件
}

// 公开方法
func (c *EthClient) GetLatestBlockNumber() (int64, error)
func (c *EthClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error)
func (c *EthClient) GetTransactionReceipt(txHash string) (map[string]interface{}, error)

// 工具函数
func hexToDecimal(hex string) (int64, error)
func hexToDecimalDefault(hex string, defaultVal int64) int64
func hexToDecimalStr(hex string) string
func hexToIntDefault(hex string, defaultVal int) int
func weiToEthStr(wei string) string  // Wei 转 ETH（18 位小数），适配 numeric(78,18)
func logProxyInfo(chain, apiURL string)
```

> **关键点**：`NewEthClient` 不手动配置 `http.Transport{Proxy: ...}`，而是依赖 Go 标准库的 `http.DefaultTransport`（它自动读取 `HTTP_PROXY`/`HTTPS_PROXY` 环境变量）。`config.Load()` 里会把 `.env` 的代理设置 `os.Setenv` 到环境变量，所以代理能生效。

### 12.2 比特币客户端（BlockCypher REST API）

```bash
touch internal/client/btc_client.go
```

关键结构（完整 232 行请对照仓库 `internal/client/btc_client.go`）：

```go
package client

type BtcClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBtcClient 接收三个参数但硬编码使用 BlockCypher API，忽略参数
func NewBtcClient(rpcURL, rpcUser, rpcPassword string) *BtcClient {
	baseURL := "https://api.blockcypher.com/v1/btc/main"
	logBtcProxyInfo(baseURL)
	return &BtcClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *BtcClient) get(path string, target interface{}) error
func (c *BtcClient) GetLatestBlockNumber() (int64, error)
func (c *BtcClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error)
// 每个区块最多获取 20 笔交易详情（BlockCypher 免费版限制）
```

> **注意**：`NewBtcClient` 接收三个参数（`rpcURL, rpcUser, rpcPassword`）但**硬编码使用 BlockCypher API**，忽略这些参数。这是历史遗留设计，保持签名兼容。`.env` 里的 `BTC_RPC_URL` 实际不被使用。

### 12.3 Solana 客户端（JSON-RPC）

```bash
touch internal/client/sol_client.go
```

关键结构（完整 317 行请对照仓库 `internal/client/sol_client.go`）：

```go
package client

type SolClient struct {
	rpcURL     string
	httpClient *http.Client
}

func NewSolClient(rpcURL string) *SolClient

// call 复用 eth_client.go 里定义的 jsonRPCRequest/jsonRPCResponse（同包）
func (c *SolClient) call(method string, params ...interface{}) (json.RawMessage, error)

func (c *SolClient) GetLatestBlockNumber() (int64, error)  // 调用 getBlockHeight
func (c *SolClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error)
// 调用 getBlock，参数含 encoding/transactionDetails/commitment/maxSupportedTransactionVersion
// lamports → SOL（除以 1e9，9 位小数）
// 从 accountKeys + pre/post balances 推导 from/to/value
// Slot 字段存到 model.Block.Slot（指针）
```

编译检查：

```bash
go build ./internal/client/
```

<details>
<summary>为什么本指南第 12 章不贴完整代码？</summary>

三个 client 文件合计 980 行，且包含大量 JSON 结构体定义和字段映射逻辑。全贴进指南会让文件膨胀到难以维护，且复制时容易出错。**最可靠的做法是直接从项目仓库复制这三个文件**——它们已经验证过能编译、能跑通真实链上数据。本指南只展示结构签名，让你理解设计。

</details>

---

## 第 13 章：区块同步服务 —— sync 包

Sync Worker 是三个持续运行的 goroutine，每个负责一条链：定时从区块链节点拉取最新区块，封装后发到 Kafka。

### 13.1 以太坊同步 Worker

```bash
touch internal/service/sync/eth_sync.go
```

完整代码（173 行）请对照仓库 `internal/service/sync/eth_sync.go`。关键结构：

```go
package sync

type EthSyncWorker struct {
	client   *client.EthClient
	producer *mq.Producer
	interval time.Duration
}

func NewEthSyncWorker(ethClient *client.EthClient, producer *mq.Producer, syncInterval int) *EthSyncWorker

func (w *EthSyncWorker) Run(ctx context.Context) error
// Run: 启动时立即 sync 一次 -> time.NewTicker(interval) -> for { select ctx.Done/ticker.C -> sync }

func (w *EthSyncWorker) sync(ctx context.Context) error
// sync: GetLatestBlockNumber -> GetBlockByNumber -> fillReceipts(并发) -> 封装 BlockMessage -> producer.Send

func (w *EthSyncWorker) fillReceipts(ctx context.Context, txs []model.Transaction)
// fillReceipts: 信号量限并发 10，goroutine 并发调用 GetTransactionReceipt
//                填充 txs[idx].GasUsed 和 txs[idx].Status
```

### 13.2 比特币同步 Worker

```bash
touch internal/service/sync/btc_sync.go
```

完整代码（114 行）请对照仓库 `internal/service/sync/btc_sync.go`。关键结构：

```go
package sync

type BtcSyncWorker struct {
	client   *client.BtcClient
	producer *mq.Producer
	interval time.Duration
}

func NewBtcSyncWorker(btcClient *client.BtcClient, producer *mq.Producer, syncInterval int) *BtcSyncWorker
func (w *BtcSyncWorker) Run(ctx context.Context) error
func (w *BtcSyncWorker) sync(ctx context.Context) error
// sync: GetLatestBlockNumber -> GetBlockByNumber -> 封装 BlockMessage -> producer.Send
```

### 13.3 Solana 同步 Worker

```bash
touch internal/service/sync/sol_sync.go
```

完整代码（138 行）请对照仓库 `internal/service/sync/sol_sync.go`。关键结构（**带重试和指数退避**）：

```go
package sync

type SolSyncWorker struct {
	client   *client.SolClient
	producer *mq.Producer
	interval time.Duration
}

func NewSolSyncWorker(solClient *client.SolClient, producer *mq.Producer, syncInterval int) *SolSyncWorker
func (w *SolSyncWorker) Run(ctx context.Context) error
func (w *SolSyncWorker) sync(ctx context.Context) error
// sync: maxRetries=3，指数退避 1s/2s/4s
//        GetLatestBlockNumber -> GetBlockByNumber -> 封装 BlockMessage -> producer.Send
//        失败 continue 重试，三次都失败返回 fmt.Errorf
```

编译检查：

```bash
go build ./internal/service/sync/
```

---


## 第 14 章：区块处理服务 —— processor 包

Block Processor 是 Kafka 的另一端：消费消息，解析数据，写入数据库。

```bash
touch internal/service/processor/block_processor.go
```

写入 `internal/service/processor/block_processor.go`（142 行，**使用接口定义依赖，便于单元测试 mock**）：

```go
// ============================================================
// Package processor 提供区块处理器服务
// ============================================================
// 从 Kafka 消费原始区块数据，解析后写入 PostgreSQL。
//
// 处理流程：
//   1. 从 Kafka 接收 BlockMessage
//   2. 解析区块数据（反序列化 JSON）
//   3. 解析交易数据
//   4. 保存区块到数据库
//   5. 设置交易的外键关联
//   6. 批量保存交易到数据库
//
// Go 语言基础知识:
//   - json.Marshal：将结构体序列化为 JSON 字节
//   - json.Unmarshal：将 JSON 字节反序列化为结构体
//   - interface{}：空接口，可以持有任意类型的值
//   - map[string]interface{}：键为字符串的 map，值可以是任意类型
//   - range：遍历切片或 map
//   - append：向切片追加元素
//   - for i := range transactions：遍历切片，i 是索引
// ============================================================
package processor

import (
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串

	"blockexplore/internal/model" // 数据模型
	"blockexplore/internal/mq"    // Kafka 消息队列
	"blockexplore/pkg/logger"     // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// 依赖接口定义（用于解耦和单元测试 mock）
// ============================================================

// BlockWriter 区块写入接口
type BlockWriter interface {
	CreateSingle(block *model.Block) error
}

// TxWriter 交易写入接口
type TxWriter interface {
	Create(txs []model.Transaction) error
}

// ============================================================
// BlockProcessor 区块处理器
// ============================================================
// 从 Kafka 消费消息，解析区块和交易数据，写入数据库
type BlockProcessor struct {
	blockRepo BlockWriter // 区块数据访问层
	txRepo    TxWriter    // 交易数据访问层
}

// ============================================================
// NewBlockProcessor 创建区块处理器实例
// ============================================================
func NewBlockProcessor(blockRepo BlockWriter, txRepo TxWriter) *BlockProcessor {
	return &BlockProcessor{
		blockRepo: blockRepo,
		txRepo:    txRepo,
	}
}

// ============================================================
// Handle 方法：处理从 Kafka 消费到的区块消息
// ============================================================
// 参数 msg：Kafka 消息（包含链标识、区块高度、原始数据）
// 这个方法会被 Consumer 调用，作为消息处理函数
func (p *BlockProcessor) Handle(msg mq.BlockMessage) error {
	logger.Info("开始处理区块消息",
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
	)

	// ============================================================
	// 第 1 步：将原始数据反序列化为 map
	// ============================================================
	// msg.Data 是 interface{} 类型，需要先序列化为 JSON，再反序列化为 map
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("序列化区块数据失败: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("解析区块数据失败: %w", err)
	}

	// ============================================================
	// 第 2 步：解析区块数据
	// ============================================================
	// data["block"] 是 interface{} 类型，需要再次序列化/反序列化
	blockData, err := json.Marshal(data["block"])
	if err != nil {
		return fmt.Errorf("解析区块失败: %w", err)
	}

	var block model.Block
	if err := json.Unmarshal(blockData, &block); err != nil {
		return fmt.Errorf("反序列化区块失败: %w", err)
	}

	// ============================================================
	// 第 3 步：解析交易数据
	// ============================================================
	txData, err := json.Marshal(data["transactions"])
	if err != nil {
		return fmt.Errorf("解析交易失败: %w", err)
	}

	var transactions []model.Transaction
	if err := json.Unmarshal(txData, &transactions); err != nil {
		return fmt.Errorf("反序列化交易失败: %w", err)
	}

	// ============================================================
	// 第 4 步：保存区块到数据库
	// ============================================================
	if err := p.blockRepo.CreateSingle(&block); err != nil {
		return fmt.Errorf("保存区块失败: %w", err)
	}

	// ============================================================
	// 第 5 步：设置交易的区块 ID（外键关联）
	// ============================================================
	// block.ID 是数据库自动生成的主键
	// 保存区块后，GORM 会自动填充 ID 字段
	// for i := range transactions 遍历切片，i 是索引
	// 我们需要修改切片元素，所以使用索引访问
	for i := range transactions {
		transactions[i].BlockID = block.ID // 设置外键
	}

	// ============================================================
	// 第 6 步：批量保存交易到数据库
	// ============================================================
	if len(transactions) > 0 {
		if err := p.txRepo.Create(transactions); err != nil {
			return fmt.Errorf("保存交易失败: %w", err)
		}
	}

	logger.Info("区块处理完成",
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
		zap.Int("tx_count", len(transactions)),
	)

	return nil
}
```

> **为什么用接口而不是具体结构体？** `BlockProcessor` 依赖 `BlockWriter` 和 `TxWriter` 接口，而非 `*repository.BlockRepo`。这样单元测试时可以传入 mock 实现（不连真实数据库），符合 Go 的"隐式接口"设计哲学。`*repository.BlockRepo` 天然满足 `BlockWriter` 接口（它有 `CreateSingle` 方法），无需修改 repository 代码。

编译检查：

```bash
go build ./internal/service/processor/
```

---

## 第 15 章：价格服务 —— price 包

```bash
touch internal/service/price/price_service.go
```

完整代码（359 行）请对照仓库 `internal/service/price/price_service.go`。关键结构：

```go
package price

type PriceService struct {
	priceRepo  *repository.PriceRepo
	cache      *cache.RedisClient
	apiURL     string
	httpClient *http.Client
}

// 构造函数接收三个参数（注意 cache 是第二个参数）
func NewPriceService(priceRepo *repository.PriceRepo, redisClient *cache.RedisClient, apiURL string) *PriceService

type PriceResponse struct {
	Chain     string  `json:"chain"`
	Symbol    string  `json:"symbol"`
	PriceUSD  float64 `json:"price_usd"`
	Timestamp int64   `json:"timestamp"`
}

type PriceHistoryResponse struct {
	Chain  string               `json:"chain"`
	Symbol string               `json:"symbol"`
	Prices []model.PriceHistory `json:"prices"`
}

// 各链对应的 CoinGecko 代币 ID 和符号
var chainCoinIDs = map[string]string{"eth": "ethereum", "btc": "bitcoin", "sol": "solana"}
var chainSymbols = map[string]string{"eth": "ETH", "btc": "BTC", "sol": "SOL"}

// 公开方法
func (s *PriceService) GetCurrentPrice(chain string) (*PriceResponse, error)
// Cache-Aside: 缓存键 "price:current:{chain}"，60s TTL
//              缓存未命中 -> priceRepo.GetLatestPrice -> 没有则 fetchPriceFromAPI

func (s *PriceService) GetPriceHistory(chain string, startTime, endTime int64, limit int) (*PriceHistoryResponse, error)
// 直接查 priceRepo.GetPriceHistory

func (s *PriceService) SyncPrices() error
// 遍历 chainCoinIDs，对每条链：fetchPriceWithRetry(coinID, 3) -> priceRepo.Create -> 更新缓存
// 每条链之间 sleep 2s（避免 CoinGecko 限流 429）

// 内部方法
func (s *PriceService) fetchPriceWithRetry(coinID string, maxRetries int) (float64, error)
// 指数退避：2s, 4s, 8s

func (s *PriceService) fetchPriceFromAPI(chain string) (*PriceResponse, error)
func (s *PriceService) fetchPrice(coinID string) (float64, error)
// GET {apiURL}/simple/price?ids={coinID}&vs_currencies=usd
// 兼容 float64 和 string 两种返回格式
```

> **关键点**：`NewPriceService` 接收**三个参数**（`priceRepo, redisClient, apiURL`），不是两个。`cmd/price-api/main.go` 和 `cmd/query-api/main.go` 都按这个签名调用。`price-api` 还用 `github.com/robfig/cron/v3` 定时调用 `SyncPrices()`。

编译检查：

```bash
go build ./internal/service/price/
```

---

## 第 16 章：查询服务 —— query 包

查询服务采用 Cache-Aside 模式：先查 Redis 缓存，未命中再查数据库，结果写入缓存。

<details>
<summary>什么是 Cache-Aside 模式？</summary>

```
读请求 → 查 Redis
         ├── 命中(cache hit) → 直接返回
         └── 未命中(cache miss) → 查 PostgreSQL → 写入 Redis → 返回
```

</details>

```bash
touch internal/service/query/query_service.go
```

写入 `internal/service/query/query_service.go`（229 行，**使用接口定义依赖，便于 mock**）：

```go
// ============================================================
// Package query 提供查询服务
// ============================================================
// 封装区块和交易的查询逻辑，支持 Redis 缓存。
//
// 缓存策略（Cache-Aside 模式）：
//   1. 查询时先读 Redis 缓存
//   2. 缓存命中则直接返回
//   3. 缓存未命中则查询数据库
//   4. 查询结果写入缓存（设置过期时间）
//   5. 返回结果
//
// 为什么使用缓存？
//   - 减少数据库压力：热点数据从缓存读取，不走数据库
//   - 提高响应速度：Redis 内存读取比数据库快 10-100 倍
//   - 支持高并发：Redis 单机支持 10 万+ QPS
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - context.Context：上下文，用于超时控制
//   - time.Duration：时间间隔类型
//   - fmt.Sprintf：格式化字符串
//   - error：Go 的错误类型
//   - interface{}：空接口，可以持有任意类型的值
// ============================================================
package query

import (
	"context"    // 上下文
	"fmt"        // 格式化字符串
	"time"       // 时间处理

	"blockexplore/internal/model"       // 数据模型
	"blockexplore/pkg/logger"          // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// 依赖接口定义（用于解耦和单元测试 mock）
// ============================================================
// 通过接口而非具体结构体声明依赖，遵循 Go 的"隐式接口"设计。
// *repository.BlockRepo / *repository.TxRepo / *cache.RedisClient
// 都天然满足这些接口，无需修改它们。

// BlockRepository 区块数据访问接口
type BlockRepository interface {
	GetList(chain string, page, pageSize int) ([]model.Block, int64, error)
	GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error)
}

// TxRepository 交易数据访问接口
type TxRepository interface {
	GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error)
	GetByHash(chain, txHash string) (*model.Transaction, error)
	GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error)
}

// Cacher 缓存接口
type Cacher interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

// ============================================================
// QueryService 查询服务
// ============================================================
// 提供区块和交易的查询功能，优先读取 Redis 缓存
type QueryService struct {
	blockRepo BlockRepository // 区块数据访问层
	txRepo    TxRepository    // 交易数据访问层
	cache     Cacher          // Redis 缓存客户端
}

// ============================================================
// NewQueryService 创建查询服务实例
// ============================================================
// 参数接受接口类型，可传入真实的 *repository.BlockRepo 等，也可传入测试 mock
func NewQueryService(blockRepo BlockRepository, txRepo TxRepository, redisClient Cacher) *QueryService {
	return &QueryService{
		blockRepo: blockRepo,
		txRepo:    txRepo,
		cache:     redisClient,
	}
}

// ============================================================
// BlockListResponse 区块列表响应
// ============================================================
type BlockListResponse struct {
	Chain      string        `json:"chain"`      // 链标识
	Blocks     []model.Block `json:"blocks"`      // 区块列表
	Pagination Pagination    `json:"pagination"`  // 分页信息
}

// ============================================================
// Pagination 分页信息
// ============================================================
type Pagination struct {
	Page     int   `json:"page"`      // 当前页码
	PageSize int   `json:"page_size"` // 每页数量
	Total    int64 `json:"total"`     // 总记录数
}

// ============================================================
// TxListResponse 交易列表响应
// ============================================================
type TxListResponse struct {
	Chain        string              `json:"chain"`        // 链标识
	Transactions []model.Transaction `json:"transactions"` // 交易列表
	Pagination   Pagination          `json:"pagination"`   // 分页信息
}

// ============================================================
// GetBlockList 方法：获取区块列表（分页）
// ============================================================
// 实现 Cache-Aside 缓存模式
func (s *QueryService) GetBlockList(chain string, page, pageSize int) (*BlockListResponse, error) {
	// 构建缓存键
	// 格式: "blocks:eth:1:20" 表示以太坊第 1 页每页 20 条
	cacheKey := fmt.Sprintf("blocks:%s:%d:%d", chain, page, pageSize)

	// 尝试从缓存读取
	var result BlockListResponse
	if s.cache != nil {
		// s.cache.Get 尝试从 Redis 读取并反序列化
		if err := s.cache.Get(context.Background(), cacheKey, &result); err == nil {
			logger.Debug("命中区块列表缓存", zap.String("key", cacheKey))
			return &result, nil // 缓存命中，直接返回
		}
	}

	// 缓存未命中，查询数据库
	blocks, total, err := s.blockRepo.GetList(chain, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 构建响应
	result = BlockListResponse{
		Chain:  chain,
		Blocks: blocks,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}

	// 写入缓存（过期时间 30 秒）
	// 30 * time.Second 表示 30 秒后自动删除
	if s.cache != nil {
		if err := s.cache.Set(context.Background(), cacheKey, &result, 30*time.Second); err != nil {
			logger.Warn("写入区块列表缓存失败", zap.Error(err))
			// 缓存写入失败不影响业务，只记录日志
		}
	}

	return &result, nil
}

// ============================================================
// GetBlockDetail 方法：获取区块详情
// ============================================================
func (s *QueryService) GetBlockDetail(chain string, blockNumber int64) (*model.Block, error) {
	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("block:%s:%d", chain, blockNumber)
	var block model.Block
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), cacheKey, &block); err == nil {
			return &block, nil
		}
	}

	// 查询数据库
	blockPtr, err := s.blockRepo.GetByChainAndNumber(chain, blockNumber)
	if err != nil {
		return nil, err
	}

	// 写入缓存（60 秒过期）
	if s.cache != nil {
		s.cache.Set(context.Background(), cacheKey, blockPtr, 60*time.Second)
	}

	return blockPtr, nil
}

// ============================================================
// GetBlockTransactions 方法：获取区块内的交易列表
// ============================================================
func (s *QueryService) GetBlockTransactions(chain string, blockNumber int64, page, pageSize int) (*TxListResponse, error) {
	txs, total, err := s.txRepo.GetByBlockNumber(chain, blockNumber, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &TxListResponse{
		Chain:        chain,
		Transactions: txs,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

// ============================================================
// GetTransactionDetail 方法：获取交易详情
// ============================================================
func (s *QueryService) GetTransactionDetail(chain, txHash string) (*model.Transaction, error) {
	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("tx:%s:%s", chain, txHash)
	var tx model.Transaction
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), cacheKey, &tx); err == nil {
			return &tx, nil
		}
	}

	// 查询数据库
	txPtr, err := s.txRepo.GetByHash(chain, txHash)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if s.cache != nil {
		s.cache.Set(context.Background(), cacheKey, txPtr, 60*time.Second)
	}

	return txPtr, nil
}

// ============================================================
// GetAddressTransactions 方法：获取地址的交易历史
// ============================================================
func (s *QueryService) GetAddressTransactions(chain, address string, page, pageSize int) (*TxListResponse, error) {
	txs, total, err := s.txRepo.GetByAddress(chain, address, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &TxListResponse{
		Chain:        chain,
		Transactions: txs,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}
```

编译检查：

```bash
go build ./internal/service/query/
```

---


## 第 17 章：HTTP 处理层 —— handler 包

Handler 做三件事：解析 HTTP 参数 → 调 Service → 返回 JSON。

### 17.1 区块 Handler

```bash
touch internal/handler/block_handler.go
```

完整代码（147 行）请对照仓库 `internal/handler/block_handler.go`。关键结构：

```go
package handler

type BlockHandler struct {
	queryService *query.QueryService
}

func NewBlockHandler(queryService *query.QueryService) *BlockHandler

// GET /api/v1/blocks?chain=eth&page=1&page_size=20
func (h *BlockHandler) GetBlockList(c *gin.Context)
// 默认值：chain=eth, page=1, page_size=20；page_size 上限 50
// 错误：CodeDBError(501) -> 500

// GET /api/v1/blocks/:block_number?chain=eth
func (h *BlockHandler) GetBlockDetail(c *gin.Context)
// block_number 解析失败 -> CodeBadRequest(400) -> 400
// 未找到 -> CodeNotFound(404) -> 404

// GET /api/v1/blocks/:block_number/transactions?chain=eth&page=1&page_size=20
func (h *BlockHandler) GetBlockTransactions(c *gin.Context)
```

### 17.2 交易 Handler

```bash
touch internal/handler/tx_handler.go
```

完整代码（91 行）请对照仓库 `internal/handler/tx_handler.go`。关键结构：

```go
package handler

type TxHandler struct {
	queryService *query.QueryService
}

func NewTxHandler(queryService *query.QueryService) *TxHandler

// GET /api/v1/transactions/:hash?chain=eth
func (h *TxHandler) GetTransactionDetail(c *gin.Context)
// 未找到 -> CodeNotFound(404) -> 404

// GET /api/v1/addresses/:address/transactions?chain=eth&page=1&page_size=20
func (h *TxHandler) GetAddressTransactions(c *gin.Context)
// 错误 -> CodeDBError(501) -> 500
```

### 17.3 搜索 Handler

```bash
touch internal/handler/search_handler.go
```

完整代码（75 行）请对照仓库 `internal/handler/search_handler.go`。关键结构：

```go
package handler

type SearchHandler struct {
	searchRepo *repository.SearchRepo
}

func NewSearchHandler(searchRepo *repository.SearchRepo) *SearchHandler

// GET /api/v1/search?q=keyword
func (h *SearchHandler) Search(c *gin.Context)
// q 为空 -> ErrorWithMsg(CodeBadRequest, "搜索关键词不能为空") -> 400
// result == nil -> ErrorWithMsg(CodeNotFound, "未找到匹配结果") -> 404
// 调用 searchRepo.Search(keyword) 返回 *SearchResult
```

### 17.4 价格 Handler

```bash
touch internal/handler/price_handler.go
```

完整代码（88 行）请对照仓库 `internal/handler/price_handler.go`。关键结构：

```go
package handler

type PriceHandler struct {
	priceService *price.PriceService  // 注意：依赖 PriceService，不是 PriceRepo
}

func NewPriceHandler(priceService *price.PriceService) *PriceHandler

// GET /api/v1/price/:chain
func (h *PriceHandler) GetCurrentPrice(c *gin.Context)
// 错误 -> CodeInternalError(500) -> 500

// GET /api/v1/price/:chain/history?start_time=&end_time=&limit=100
func (h *PriceHandler) GetPriceHistory(c *gin.Context)
// limit 默认 100，上限 1000
// 错误 -> CodeDBError(501) -> 500
```

> **关键 bug 修复点**：`cmd/query-api/main.go` 中 `priceHandler` **必须传入真实的 `PriceService`**，不能传 `nil`。否则前端访问 `/api/v1/price/*`（next.config.js 把所有 `/api/v1/*` 代理到 query-api）会让 query-api **panic 崩溃**。本指南第 20.1 节会展示正确的组装方式。

编译检查：

```bash
go build ./internal/handler/
```

---

## 第 18 章：中间件 —— middleware 包

### 18.1 CORS 中间件

```bash
touch internal/middleware/cors.go
```

写入 `internal/middleware/cors.go`（33 行）：

```go
// Package middleware 提供 Gin 中间件
// 中间件在请求到达 Handler 之前/之后执行，用于通用处理
package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 跨域资源共享中间件
// 允许前端跨域访问 API
// 浏览器出于安全考虑，会限制跨域请求，此中间件添加必要的响应头
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 允许所有来源访问（生产环境应该限制为具体域名）
		c.Header("Access-Control-Allow-Origin", "*")
		// 允许的 HTTP 方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// 允许的请求头
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		// 允许暴露的响应头
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		// 预检请求缓存时间（秒）
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求（OPTIONS 方法）
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // 返回 204 No Content
			return
		}

		c.Next() // 继续处理下一个中间件或 Handler
	}
}
```

### 18.2 Request ID 中间件

```bash
touch internal/middleware/request_id.go
```

写入 `internal/middleware/request_id.go`（29 行）：

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 请求 ID 中间件
// 为每个请求生成唯一的 UUID，用于链路追踪和日志关联
// 如果请求头中已包含 X-Request-ID，则复用该值
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取请求 ID
		requestID := c.GetHeader("X-Request-ID")

		// 如果没有，生成新的 UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 存储到上下文中，供后续 Handler 使用
		c.Set("request_id", requestID)

		// 添加到响应头
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
```

### 18.3 限流中间件

```bash
touch internal/middleware/ratelimit.go
```

写入 `internal/middleware/ratelimit.go`（121 行，**令牌桶算法，按 IP 限流**）：

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"blockexplore/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// RateLimiter 令牌桶限流器
// 使用内存实现的简单令牌桶算法
// 生产环境建议使用 Redis 实现分布式限流
type RateLimiter struct {
	rate       int           // 每秒生成的令牌数
	bucketSize int           // 桶容量
	buckets    map[string]*bucket // 客户端令牌桶
	mu         sync.Mutex    // 互斥锁
}

// bucket 单个令牌桶
type bucket struct {
	tokens    int       // 当前令牌数
	lastTime  time.Time // 上次更新时间
}

// NewRateLimiter 创建限流器实例
// rate: 每秒允许的请求数
// bucketSize: 突发请求容量
func NewRateLimiter(rate, bucketSize int) *RateLimiter {
	rl := &RateLimiter{
		rate:       rate,
		bucketSize: bucketSize,
		buckets:    make(map[string]*bucket),
	}

	// 定期清理过期的令牌桶（每分钟）
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// RateLimit 限流中间件
// 根据客户端 IP 进行限流
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端 IP 作为限流 key
		key := c.ClientIP()

		// 检查是否有可用令牌
		if !rl.allow(key) {
			c.JSON(http.StatusTooManyRequests, errcode.ErrorWithMsg(
				http.StatusTooManyRequests,
				"请求过于频繁，请稍后再试",
				c.GetString("request_id"),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow 检查是否允许请求
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		// 首次请求，创建新的令牌桶
		rl.buckets[key] = &bucket{
			tokens:   rl.bucketSize - 1, // 消耗一个令牌
			lastTime: time.Now(),
		}
		return true
	}

	// 计算应该添加的令牌数
	now := time.Now()
	elapsed := now.Sub(b.lastTime)
	tokensToAdd := int(elapsed.Seconds()) * rl.rate

	// 更新令牌数（不超过桶容量）
	b.tokens += tokensToAdd
	if b.tokens > rl.bucketSize {
		b.tokens = rl.bucketSize
	}
	b.lastTime = now

	// 检查是否有可用令牌
	if b.tokens <= 0 {
		return false
	}

	// 消耗一个令牌
	b.tokens--
	return true
}

// cleanup 清理长时间未使用的令牌桶
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastTime.Before(threshold) {
			delete(rl.buckets, key)
		}
	}
}
```

编译检查：

```bash
go build ./internal/middleware/
```

---

## 第 19 章：路由注册 —— router 包

```bash
touch internal/router/router.go
```

写入 `internal/router/router.go`（73 行）：

```go
// Package router 提供路由注册
// 使用 Gin 框架的路由组管理所有 API 路由
package router

import (
	"blockexplore/internal/handler"
	"blockexplore/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
// 注册所有 API 路由和中间件
func Setup(
	blockHandler *handler.BlockHandler,
	txHandler *handler.TxHandler,
	searchHandler *handler.SearchHandler,
	priceHandler *handler.PriceHandler,
) *gin.Engine {
	// 创建 Gin 引擎（生产模式）
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 注册全局中间件
	r.Use(middleware.RequestID()) // 请求 ID（最先执行）
	r.Use(middleware.CORS())      // 跨域支持
	r.Use(gin.Recovery())        // panic 恢复

	// 创建限流器（每秒 100 请求，突发容量 200）
	limiter := middleware.NewRateLimiter(100, 200)
	r.Use(limiter.RateLimit())

	// 健康检查接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 区块相关接口
		blocks := v1.Group("/blocks")
		{
			blocks.GET("", blockHandler.GetBlockList)                              // 区块列表
			blocks.GET("/:block_number", blockHandler.GetBlockDetail)              // 区块详情
			blocks.GET("/:block_number/transactions", blockHandler.GetBlockTransactions) // 区块内交易
		}

		// 交易相关接口
		transactions := v1.Group("/transactions")
		{
			transactions.GET("/:hash", txHandler.GetTransactionDetail) // 交易详情
		}

		// 地址相关接口
		addresses := v1.Group("/addresses")
		{
			addresses.GET("/:address/transactions", txHandler.GetAddressTransactions) // 地址交易历史
		}

		// 搜索接口
		v1.GET("/search", searchHandler.Search)

		// 价格接口
		price := v1.Group("/price")
		{
			price.GET("/:chain", priceHandler.GetCurrentPrice)        // 当前价格
			price.GET("/:chain/history", priceHandler.GetPriceHistory) // 价格历史
		}
	}

	return r
}
```

> **注意**：`router.Setup` 注册了所有路由（包括 price 和 search）。`query-api` 会传入全部 4 个 handler，所以它本身就能服务所有 `/api/v1/*` 路由。`search-api` 和 `price-api` 是独立部署的备用服务（见第 20 章）。

编译检查：

```bash
go build ./internal/router/
```

---

## 第 20 章：七个微服务入口 —— cmd/

每个 `cmd/<服务名>/main.go` 都是一个独立的可执行程序。它们遵循相同的模板：

```
1. 加载配置
2. 初始化日志
3. 连接基础设施（DB/Redis/Kafka）
4. 创建各层实例（new → new → new）
5. 启动服务
```

### 20.1 query-api（查询 API，端口 8080）

```bash
touch cmd/query-api/main.go
```

完整代码请对照仓库 `cmd/query-api/main.go`。**关键修复点**：`priceHandler` 必须传入真实的 `PriceService`（不能是 nil）：

```go
package main

import (
	"fmt"

	"blockexplore/internal/config"
	"blockexplore/internal/handler"
	"blockexplore/internal/repository"
	"blockexplore/internal/router"
	"blockexplore/internal/service/price"  // ← 必须导入
	"blockexplore/internal/service/query"
	"blockexplore/pkg/cache"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("query-api 服务启动中...")

	// 连接数据库
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	logger.Info("数据库连接成功")

	// 初始化 Redis
	cache.Init(cfg.Redis)
	redisClient := cache.GetClient()

	// 创建 Repository
	blockRepo := repository.NewBlockRepo(db)
	txRepo := repository.NewTxRepo(db)
	priceRepo := repository.NewPriceRepo(db)  // ← 必须创建

	// 创建 Service
	queryService := query.NewQueryService(blockRepo, txRepo, redisClient)
	priceService := price.NewPriceService(priceRepo, redisClient, cfg.Price.APIURL)  // ← 必须创建

	// 创建 Handler
	blockHandler := handler.NewBlockHandler(queryService)
	txHandler := handler.NewTxHandler(queryService)
	searchHandler := handler.NewSearchHandler(repository.NewSearchRepo(db))
	priceHandler := handler.NewPriceHandler(priceService)  // ← 传入真实 service，不能是 nil

	// 注册路由
	r := router.Setup(blockHandler, txHandler, searchHandler, priceHandler)

	// 启动
	addr := fmt.Sprintf(":%d", cfg.Server.QueryAPIPort)
	logger.Info("query-api 已启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("query-api 启动失败", zap.Error(err))
	}
}
```

> **这个修复至关重要**：`next.config.js` 把所有 `/api/v1/*`（包括 `/api/v1/price/*`）都代理到 `query-api:8080`。如果 `priceHandler` 是 nil，访问价格接口会 panic。本指南版本已修复此问题。

### 20.2 eth-sync-worker（以太坊同步，无端口）

```bash
touch cmd/eth-sync-worker/main.go
```

完整代码请对照仓库 `cmd/eth-sync-worker/main.go`。关键结构：

```go
package main

func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	ethClient := client.NewEthClient(cfg.ETH.RPCURL)
	producer := mq.NewETHProducer(cfg.Kafka)
	defer producer.Close()

	worker := sync.NewEthSyncWorker(ethClient, producer, cfg.ETH.SyncInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 优雅关闭：监听 SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigChan; cancel() }()

	worker.Run(ctx)  // 阻塞运行
}
```

### 20.3 btc-sync-worker（比特币同步）

```bash
touch cmd/btc-sync-worker/main.go
```

完整代码请对照仓库 `cmd/btc-sync-worker/main.go`。关键结构：

```go
func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	// 注意：NewBtcClient 接收三个参数（签名兼容），但内部硬编码用 BlockCypher
	btcClient := client.NewBtcClient(cfg.BTC.RPCURL, cfg.BTC.RPCUser, cfg.BTC.RPCPassword)
	producer := mq.NewBTCProducer(cfg.Kafka)
	defer producer.Close()

	worker := sync.NewBtcSyncWorker(btcClient, producer, cfg.BTC.SyncInterval)
	// ... 同 eth-sync-worker 的优雅关闭模式
	worker.Run(ctx)
}
```

### 20.4 sol-sync-worker（Solana 同步）

```bash
touch cmd/sol-sync-worker/main.go
```

完整代码请对照仓库 `cmd/sol-sync-worker/main.go`。关键结构：

```go
func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	solClient := client.NewSolClient(cfg.SOL.RPCURL)
	producer := mq.NewSOLProducer(cfg.Kafka)
	defer producer.Close()

	worker := sync.NewSolSyncWorker(solClient, producer, cfg.SOL.SyncInterval)
	// ... 同 eth-sync-worker 的优雅关闭模式
	worker.Run(ctx)
}
```

### 20.5 block-processor（Kafka 消费 + 入库）

```bash
touch cmd/block-processor/main.go
```

完整代码请对照仓库 `cmd/block-processor/main.go`。关键结构：

```go
func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	// ... 设置连接池

	// 创建处理器
	blockRepo := repository.NewBlockRepo(db)
	txRepo := repository.NewTxRepo(db)
	blockProcessor := processor.NewBlockProcessor(blockRepo, txRepo)

	// 创建三个 Topic 的消费者
	topics := []string{cfg.Kafka.ETHTopic, cfg.Kafka.RTCTopic, cfg.Kafka.SOLTopic}
	consumers := make([]*mq.Consumer, 0, len(topics))
	for _, topic := range topics {
		c := mq.NewConsumer(cfg.Kafka.Brokers, topic, cfg.Kafka.ConsumerGroup)
		consumers = append(consumers, c)
	}
	defer func() {
		for _, c := range consumers { c.Close() }
	}()

	// 优雅关闭 + 并发消费
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// ... signal.Notify

	mq.ConsumeAll(ctx, consumers, blockProcessor.Handle)  // 阻塞
}
```

> **注意**：topics 列表里用的是 `cfg.Kafka.RTCTopic`（字段名叫 RTC，实际存的是 BTC 的 topic）。这是历史遗留命名，全项目一致使用。

### 20.6 search-api（搜索 API，端口 8081）

```bash
touch cmd/search-api/main.go
```

完整代码请对照仓库 `cmd/search-api/main.go`。关键结构：

```go
func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	// ... 设置连接池

	searchHandler := handler.NewSearchHandler(repository.NewSearchRepo(db))

	// search-api 用自己的 gin 引擎（不走 router.Setup，因为只需要 /search 路由）
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())
	// 注意：search-api 没有挂限流中间件

	r.GET("/api/v1/search", searchHandler.Search)

	r.Run(fmt.Sprintf(":%d", cfg.Server.SearchAPIPort))
}
```

### 20.7 price-api（价格 API，端口 8082）

```bash
touch cmd/price-api/main.go
```

完整代码请对照仓库 `cmd/price-api/main.go`。关键结构（**用 cron 定时同步价格**）：

```go
func main() {
	cfg := config.Load()
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	// ... 设置连接池

	cache.Init(cfg.Redis)
	redisClient := cache.GetClient()

	priceRepo := repository.NewPriceRepo(db)
	priceService := price.NewPriceService(priceRepo, redisClient, cfg.Price.APIURL)

	// 启动 cron 定时同步价格
	c := cron.New()
	syncInterval := fmt.Sprintf("@every %ds", cfg.Price.SyncInterval)
	c.AddFunc(syncInterval, func() {
		if err := priceService.SyncPrices(); err != nil {
			logger.Error("价格同步失败", zap.Error(err))
		}
	})
	c.Start()

	priceHandler := handler.NewPriceHandler(priceService)

	// price-api 用自己的 gin 引擎（只注册 /price 路由）
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.CORS(), gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		priceGroup := v1.Group("/price")
		{
			priceGroup.GET("/:chain", priceHandler.GetCurrentPrice)
			priceGroup.GET("/:chain/history", priceHandler.GetPriceHistory)
		}
	}

	r.Run(fmt.Sprintf(":%d", cfg.Server.PriceAPIPort))
}
```

---

## 第 21 章：编译验证 —— 让代码跑起来

### 21.1 整理依赖

```bash
cd ~/Desktop/BlockExplore
go mod tidy
```

这条命令会：
1. 下载所有 `import` 中用到的但 `go.mod` 中没有的包
2. 删除 `go.mod` 中 `import` 没用到的包
3. 更新 `go.sum`（依赖校验文件）

### 21.2 编译所有 7 个服务

```bash
go build -o /dev/null ./cmd/query-api/
go build -o /dev/null ./cmd/search-api/
go build -o /dev/null ./cmd/price-api/
go build -o /dev/null ./cmd/eth-sync-worker/
go build -o /dev/null ./cmd/btc-sync-worker/
go build -o /dev/null ./cmd/sol-sync-worker/
go build -o /dev/null ./cmd/block-processor/
```

全部通过（无输出=无错误）。

<details>
<summary>常见编译错误及解决</summary>

**错误**：`cannot find package "xxx"`
- 运行 `go mod tidy`

**错误**：`imported and not used: "xxx"`
- 删除那个 import 行，或用 `_ "xxx"` 匿名导入

**错误**：`xxx declared and not used`
- 删除变量声明，或用 `_ = xxx` 来"使用"它

**错误**：`cannot use nil as type ... in argument to ...`
- 检查是否把 nil 传给了需要真实实例的参数（如第 20.1 节的 priceHandler）

</details>

### 21.3 测试 query-api 能否启动

```bash
go run ./cmd/query-api/
```

你应该看到类似的日志输出：

```
已加载配置文件: C:/Users/xxx/Desktop/BlockExplore/.env
[配置] DB_HOST=localhost, REDIS_HOST=localhost, KAFKA_BROKERS=localhost:9092
Redis 连接成功 addr=localhost:6379
数据库连接成功
query-api 已启动 addr=:8080
```

**按 `Ctrl + C` 停止。**

如果没看到 "Redis 连接成功" 或 "数据库连接成功"：
- 检查 Docker 是否在运行：`docker compose -f docker-compose.dev.yaml ps`
- 检查 `.env` 文件中的连接信息是否正确
- 如果用了备用 PostgreSQL 端口，确认 `.env` 里 `DB_PORT` 已改

---


## 第 22 章：Next.js 前端 —— web/

### 22.1 初始化 Next.js 项目

```bash
cd ~/Desktop/BlockExplore
npx create-next-app@14 web --typescript --tailwind --eslint --app --src-dir --no-import-alias
```

<details>
<summary>这条命令做了什么？</summary>

| 参数 | 含义 |
|------|------|
| `web` | 项目文件夹名 |
| `--typescript` | 使用 TypeScript |
| `--tailwind` | 集成 Tailwind CSS |
| `--eslint` | 集成 ESLint 代码检查 |
| `--app` | 使用 App Router（Next.js 14 新模式） |
| `--src-dir` | 使用 `src/` 目录结构 |
| `--no-import-alias` | 不使用 `@/` 路径别名 |

</details>

安装过程中会问几个问题，全部用默认选项（直接回车）。

### 22.2 安装额外的依赖

```bash
cd web
npm install recharts@^2.10.0 lucide-react@^0.300.0
```

- `recharts`：价格走势图
- `lucide-react`：图标库

### 22.3 配置 next.config.js

打开 `web/next.config.js`，替换为：

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',      // standalone 模式，支持 Docker 部署
  images: {
    unoptimized: true,
  },
  trailingSlash: true,

  // Rewrites：反向代理 API 请求，避免跨域
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: 'http://query-api:8080/api/v1/:path*',
      },
    ]
  },
}

module.exports = nextConfig
```

> **关键点**：`destination` 在 **Docker 部署**时用 `http://query-api:8080`（容器服务名）。**本地开发**时需要改为 `http://localhost:8080`。本指南的 `next.config.js` 用的是 Docker 版本——本地开发时要么改这行，要么本地也用 Docker 跑 query-api。

### 22.4 全局样式

打开 `web/src/app/globals.css`，替换为（深色主题，Etherscan 风格）：

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  --bg-primary: #0d1117;
  --bg-secondary: #161b22;
  --bg-card: #1c2129;
  --text-primary: #e6edf3;
  --text-secondary: #8b949e;
  --accent-blue: #58a6ff;
  --accent-green: #3fb950;
  --border-color: #30363d;
}

body {
  background-color: var(--bg-primary);
  color: var(--text-primary);
}

@layer components {
  .card {
    @apply bg-[#1c2129] border border-[#30363d] rounded-lg p-4;
  }

  .btn {
    @apply px-4 py-2 rounded-lg bg-[#21262d] border border-[#30363d]
           text-[#c9d1d9] hover:bg-[#30363d] transition-colors
           disabled:opacity-50 disabled:cursor-not-allowed;
  }

  .hash-text {
    @apply font-mono text-[#58a6ff] truncate;
  }
}
```

### 22.5 API 客户端

```bash
mkdir -p src/lib
touch src/lib/api.ts
```

完整代码请对照仓库 `web/src/lib/api.ts`。关键结构：

```typescript
export interface Block { id, chain, block_number, block_hash, timestamp, tx_count, gas_used, gas_limit, slot? }
export interface Transaction { id, chain, tx_hash, block_number, from_addr, to_addr, value, status, timestamp }
export interface Pagination { page, page_size, total }

const BASE = '/api/v1'

export async function fetchBlocks(chain, page=1, pageSize=20)
export async function fetchBlockDetail(chain, blockNumber)
export async function fetchTransaction(chain, hash)
export async function fetchAddressTransactions(chain, address, page=1, pageSize=20)
export async function fetchPrice(chain)
```

### 22.6 创建组件

```bash
mkdir -p src/components
touch src/components/Header.tsx
touch src/components/SearchBar.tsx
touch src/components/BlockTable.tsx
touch src/components/TxTable.tsx
touch src/components/PriceCard.tsx
touch src/components/PriceChart.tsx
```

完整代码请对照仓库 `web/src/components/` 目录下的对应文件。组件清单：

| 组件 | 作用 |
|------|------|
| `Header.tsx` | 顶部导航栏 + 三链切换（eth/btc/sol） |
| `SearchBar.tsx` | 搜索框，根据输入格式跳转（区块号/0x 哈希/Solana 地址） |
| `BlockTable.tsx` | 区块列表表格（高度、哈希、交易数、时间） |
| `TxTable.tsx` | 交易列表表格（哈希、发送方、接收方） |
| `PriceCard.tsx` | 价格卡片，调用 `fetchPrice` |
| `PriceChart.tsx` | 价格走势图（用 recharts） |

### 22.7 创建页面

```bash
# App Router 动态路由目录
mkdir -p "src/app/blocks/[chain]"
mkdir -p "src/app/blocks/[chain]/[number]"
mkdir -p "src/app/tx/[chain]/[hash]"
mkdir -p "src/app/address/[chain]/[address]"
```

页面文件清单（完整代码请对照仓库 `web/src/app/`）：

| 文件 | 路由 | 作用 |
|------|------|------|
| `layout.tsx` | - | 根布局 |
| `page.tsx` | `/` | 首页（三链价格卡片 + 最新区块） |
| `blocks/[chain]/page.tsx` | `/blocks/eth/` | 区块列表（分页） |
| `blocks/[chain]/[number]/page.tsx` | `/blocks/eth/123/` | 区块详情 |
| `tx/[chain]/[hash]/page.tsx` | `/tx/eth/0x.../` | 交易详情 |
| `address/[chain]/[address]/page.tsx` | `/address/eth/0x.../` | 地址交易记录 |

### 22.8 启动前端（本地开发）

```bash
cd ~/Desktop/BlockExplore/web
npm run dev
```

你应该看到：
```
  ▲ Next.js 14.x
  - Local:        http://localhost:3000

 ✓ Ready in 2.5s
```

打开浏览器访问 `http://localhost:3000`。如果 query-api 也在运行（第 21 章），你能看到价格卡片和区块列表。

> **本地开发注意**：`next.config.js` 的 rewrites 指向 `http://query-api:8080`（Docker 服务名）。本地开发时要么：1）把 query-api 也用 Docker 跑；2）临时把 rewrites 改成 `http://localhost:8080`。**提交到 Git 时记得改回 `query-api:8080`**（Docker 部署用）。

---

## 第 23 章：Docker 容器化

### 23.1 .env.docker（Docker 环境配置）

```bash
cd ~/Desktop/BlockExplore
touch .env.docker
```

写入 `.env.docker`（62 行，与项目实际代码一致）：

```bash
# ============================================================
# BlockExplore Docker 环境配置
# 容器之间通过服务名通信（不是 localhost）
# ============================================================

APP_NAME=blockexplore
APP_ENV=production
APP_DEBUG=false

# PostgreSQL（容器内通过服务名访问）
DB_HOST=postgres
DB_PORT=5432
DB_USER=blockexplore
DB_PASSWORD=blockexplore123
DB_NAME=blockexplore
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=10

# Redis（容器内通过服务名访问）
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=100

# Kafka（容器内通过服务名访问，使用内部端口 29092）
KAFKA_BROKERS=kafka:29092
KAFKA_ETH_TOPIC=block.raw.eth
KAFKA_BTC_TOPIC=block.raw.btc
KAFKA_SOL_TOPIC=block.raw.sol
KAFKA_CONSUMER_GROUP=block-processor-group

# 代理配置（VPN 端口 7890）
HTTP_PROXY=http://host.docker.internal:7890
HTTPS_PROXY=http://host.docker.internal:7890

# 以太坊 RPC
ETH_RPC_URL=https://eth.drpc.org
ETH_SYNC_INTERVAL=12

# 比特币 RPC
BTC_RPC_URL=http://localhost:8332
BTC_RPC_USER=bitcoin
BTC_RPC_PASSWORD=bitcoin123
BTC_SYNC_INTERVAL=600

# Solana RPC
SOL_RPC_URL=https://api.mainnet-beta.solana.com
SOL_SYNC_INTERVAL=5

# 价格 API
PRICE_API_URL=https://api.coingecko.com/api/v3
PRICE_SYNC_INTERVAL=120

# API 服务端口（容器内部）
QUERY_API_PORT=8080
SEARCH_API_PORT=8081
PRICE_API_PORT=8082

LOG_LEVEL=info
LOG_FORMAT=json
```

**关键差异（对比 `.env`）**：
- `DB_HOST=postgres`（不是 localhost）
- `REDIS_HOST=redis`（不是 localhost）
- `KAFKA_BROKERS=kafka:29092`（用内部 listener 端口）
- `HTTP_PROXY=http://host.docker.internal:7890`（容器通过这个特殊域名访问宿主机代理）

### 23.2 Go 后端 Dockerfile

```bash
touch Dockerfile
```

写入 `Dockerfile`（62 行，**多阶段构建，含 GOPROXY 加速**）：

```dockerfile
# ============================================================
# BlockExplore 多阶段构建 Dockerfile
# 阶段1: 编译 Go 程序
# 阶段2: 运行时镜像（仅包含二进制文件，体积小）
# ============================================================

# ---------- 阶段1: 编译阶段 ----------
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装 Git（go mod 需要）
RUN apk add --no-cache git

# 国内构建加速：使用 goproxy.cn，并清除可能从宿主继承的代理设置
ENV GOPROXY=https://goproxy.cn,direct
ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV http_proxy=
ENV https_proxy=

# 先复制依赖文件，利用 Docker 缓存层
# 只有 go.mod/go.sum 变化时才重新下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 编译所有微服务的二进制文件
# CGO_ENABLED=0 禁用 CGO，生成静态链接的二进制文件
# -ldflags="-s -w" 去除调试信息，减小体积
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/query-api ./cmd/query-api/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/search-api ./cmd/search-api/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/price-api ./cmd/price-api/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/eth-sync-worker ./cmd/eth-sync-worker/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/btc-sync-worker ./cmd/btc-sync-worker/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/sol-sync-worker ./cmd/sol-sync-worker/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/block-processor ./cmd/block-processor/

# ---------- 阶段2: 运行阶段 ----------
FROM alpine:3.19

# 设置工作目录
WORKDIR /app

# 安装必要的运行时依赖
# ca-certificates: HTTPS 证书（访问区块链节点需要）
# tzdata: 时区数据
RUN apk add --no-cache ca-certificates tzdata

# 从编译阶段复制二进制文件
COPY --from=builder /app/bin/ /app/

# 复制配置文件（运行时通过 env_file 覆盖）
COPY .env.docker /app/.env

# 复制数据库迁移文件
COPY migrations/ /app/migrations/

# 暴露端口
# 8080: query-api
# 8081: search-api
# 8082: price-api
EXPOSE 8080 8081 8082

# 默认启动 query-api（可通过 docker-compose 覆盖）
CMD ["/app/query-api"]
```

> **为什么需要 GOPROXY 和清除代理？** Docker 构建时 `go mod download` 需要联网。如果宿主机有代理但 Docker 构建容器内代理不可达，会失败。`GOPROXY=https://goproxy.cn,direct` 用国内镜像加速，`HTTP_PROXY=` 清除从 Docker Desktop 全局配置继承的代理（避免指向不存在的代理）。

### 23.3 前端 Dockerfile

```bash
touch web/Dockerfile
```

写入 `web/Dockerfile`（38 行，三阶段构建）：

```dockerfile
# ============================================================
# BlockExplore 前端 Dockerfile
# 使用 Next.js Standalone 模式运行
# ============================================================

# 阶段1: 依赖安装
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# 阶段2: 构建
FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# 阶段3: 运行时
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
```

### 23.4 生产环境 docker-compose.yaml

```bash
touch docker-compose.yaml
```

写入 `docker-compose.yaml`（完整 11 个服务，与项目实际代码一致）：

```yaml
# ============================================================
# BlockExplore 完整 Docker Compose 配置
# ============================================================

services:
  # ==================== 基础设施 ====================

  postgres:
    image: postgres:latest
    container_name: blockexplore-postgres
    environment:
      POSTGRES_USER: blockexplore
      POSTGRES_PASSWORD: blockexplore123
      POSTGRES_DB: blockexplore
      PGDATA: /var/lib/postgresql/data/pgdata
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    networks:
      - blockexplore-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U blockexplore"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:latest
    container_name: blockexplore-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - blockexplore-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  kafka:
    image: apache/kafka:latest
    container_name: blockexplore-kafka
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_LISTENERS: INTERNAL://0.0.0.0:29092,EXTERNAL://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      KAFKA_ADVERTISED_LISTENERS: INTERNAL://kafka:29092,EXTERNAL://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT,CONTROLLER:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_INTER_BROKER_LISTENER_NAME: INTERNAL
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      KAFKA_MESSAGE_MAX_BYTES: 10485760
      KAFKA_REPLICA_FETCH_MAX_BYTES: 10485760
      CLUSTER_ID: MkU3OEVBNTcwNTJENDM2Qk
    volumes:
      - kafka_data:/var/lib/kafka/data
    networks:
      - blockexplore-net
    healthcheck:
      test: ["CMD-SHELL", "/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 > /dev/null 2>&1"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 30s

  # ==================== Go 微服务 ====================

  # query-api / search-api / block-processor：不需要外部网络，清除代理
  query-api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-query-api
    command: ["/app/query-api"]
    env_file:
      - .env.docker
    environment:
      - HTTP_PROXY=
      - HTTPS_PROXY=
      - http_proxy=
      - https_proxy=
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  search-api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-search-api
    command: ["/app/search-api"]
    env_file:
      - .env.docker
    environment:
      - HTTP_PROXY=
      - HTTPS_PROXY=
      - http_proxy=
      - https_proxy=
    ports:
      - "8081:8081"
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  block-processor:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-processor
    command: ["/app/block-processor"]
    env_file:
      - .env.docker
    environment:
      - HTTP_PROXY=
      - HTTPS_PROXY=
      - http_proxy=
      - https_proxy=
    depends_on:
      postgres:
        condition: service_healthy
      kafka:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  # price-api：需要代理访问 CoinGecko（在中国被墙）
  price-api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-price-api
    command: ["/app/price-api"]
    env_file:
      - .env.docker
    ports:
      - "8082:8082"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  # eth-sync-worker：需要代理访问以太坊 RPC
  eth-sync-worker:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-eth-sync
    command: ["/app/eth-sync-worker"]
    env_file:
      - .env.docker
    depends_on:
      kafka:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  # btc-sync-worker：需要代理访问 BlockCypher API
  btc-sync-worker:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-btc-sync
    command: ["/app/btc-sync-worker"]
    env_file:
      - .env.docker
    depends_on:
      kafka:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  # sol-sync-worker：需要代理访问 Solana RPC
  sol-sync-worker:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: blockexplore-sol-sync
    command: ["/app/sol-sync-worker"]
    env_file:
      - .env.docker
    depends_on:
      kafka:
        condition: service_healthy
    networks:
      - blockexplore-net
    restart: unless-stopped

  # ==================== 前端 ====================

  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    container_name: blockexplore-web
    ports:
      - "3000:3000"
    depends_on:
      - query-api
    networks:
      - blockexplore-net
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  kafka_data:

networks:
  blockexplore-net:
    driver: bridge
```

**关键设计点**：

1. **Kafka 双 Listener**：`INTERNAL://kafka:29092`（容器间通信）+ `EXTERNAL://localhost:9092`（宿主机调试）。`KAFKA_INTER_BROKER_LISTENER_NAME: INTERNAL` 确保 Broker 之间用内部地址。

2. **代理分离**：`query-api`/`search-api`/`block-processor` 只连内部 PG/Redis，不需要外网，所以 `environment` 里清除代理（`HTTP_PROXY=`）。`price-api` 和三个 sync-worker 需要访问外网链上节点/CoinGecko，保留 `.env.docker` 里的代理。

3. **`depends_on` + `condition: service_healthy`**：确保 PG/Redis/Kafka 健康后才启动依赖它们的 Go 服务。

4. **`${POSTGRES_PORT:-5432}`**：端口可被环境变量覆盖（应对 Windows Hyper-V 保留端口问题）。

5. **`KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"`**：sync-worker 第一次发消息时自动创建 topic，否则报 "Unknown Topic"。

---

## 第 24 章：完整运行与端到端验证

### 24.1 开发模式（全本地运行）

你需要打开 9 个 Git Bash 窗口（或终端标签）：

| 终端 | 命令 | 说明 |
|------|------|------|
| 1 | `docker compose -f docker-compose.dev.yaml up -d` | 启动基础设施 |
| 2 | `go run ./cmd/query-api/` | 查询 API |
| 3 | `go run ./cmd/search-api/` | 搜索 API |
| 4 | `go run ./cmd/price-api/` | 价格 API |
| 5 | `go run ./cmd/eth-sync-worker/` | ETH 同步 |
| 6 | `go run ./cmd/btc-sync-worker/` | BTC 同步 |
| 7 | `go run ./cmd/sol-sync-worker/` | SOL 同步 |
| 8 | `go run ./cmd/block-processor/` | Kafka 消费 + 入库 |
| 9 | `cd web && npm run dev` | 前端 |

**全部按 `Ctrl + C` 可逐个停止。**

> **本地开发注意**：前端 `next.config.js` 的 rewrites 指向 `query-api:8080`（Docker 服务名），本地跑 `npm run dev` 时需要临时改成 `localhost:8080`。

### 24.2 生产模式（一键 Docker 部署）

```bash
# 确保代理 7890 已开（Docker 构建和 sync-worker 需要）
# 在项目根目录
docker compose up -d --build

# 查看状态（等待所有服务 healthy/Up）
docker compose ps

# 查看日志
docker compose logs -f block-processor  # 只看某个服务
docker compose logs -f                  # 看全部

# 停止并清理（保留数据）
docker compose down

# 彻底清空数据（重建数据库）
docker compose down -v
```

<details>
<summary>如果 Docker 构建报 go mod download 失败</summary>

**原因**：Docker Desktop 全局配了代理（`~/.docker/config.json` 里的 `proxies`），但代理没开。

**解决**：
1. 确保代理 7890 已开（`Test-NetConnection 127.0.0.1 -Port 7890` 应返回 True）
2. Dockerfile 已设 `GOPROXY=https://goproxy.cn,direct` 和清除代理 env，通常能解决
3. 实在不行，临时编辑 `~/.docker/config.json` 删掉 `proxies` 段

</details>

<details>
<summary>如果 5432 端口被占用（Windows Hyper-V 保留）</summary>

```bash
netsh interface ipv4 show excludedportrange protocol=tcp
```
如果 5432 落在保留段内：
```bash
POSTGRES_PORT=5280 docker compose up -d --build
```
后续访问 PostgreSQL 都用 5280。

</details>

### 24.3 验证接口（端到端）

等所有容器启动 15-30 秒后（让 sync-worker 拉到数据）：

```bash
# 健康检查
curl http://localhost:8080/health
# 应返回: {"status":"ok"}

# 获取 ETH 区块列表
curl "http://localhost:8080/api/v1/blocks?chain=eth&page=1&page_size=5"
# 应返回: {"code":200,"message":"success","data":{"chain":"eth","blocks":[...],"pagination":{...}}}

# 按区块号搜索
curl "http://localhost:8080/api/v1/search?q=12345678"
# 应返回: {"code":200,...,"data":{"type":"block","data":{...}}}

# 价格
curl "http://localhost:8080/api/v1/price/eth"
# 应返回: {"code":200,...,"data":{"chain":"eth","symbol":"ETH","price_usd":xxxx,...}}

# 前端
curl http://localhost:3000/
# 应返回 HTML（状态码 200）
```

打开浏览器访问 `http://localhost:3000`，你应该看到：
- 三链价格卡片（ETH/BTC/SOL 当前价格）
- 三链最新区块列表
- 顶部搜索框可搜区块号/交易哈希/地址

**如果区块列表为空**：等 sync-worker 多跑几轮（ETH 12 秒/轮，BTC 10 分钟/轮，SOL 1-5 秒/轮）。检查日志：
```bash
docker compose logs eth-sync-worker --tail 20
docker compose logs block-processor --tail 20
```

### 24.4 服务端口速查

| 服务 | 端口 | 说明 |
|---|---|---|
| Next.js 前端 | 3000 | Web 界面 |
| query-api | 8080 | 区块/交易/地址/价格查询（网关） |
| search-api | 8081 | 搜索（独立部署备用） |
| price-api | 8082 | 价格（独立部署备用） |
| PostgreSQL | 5432 | 数据库 |
| Redis | 6379 | 缓存 |
| Kafka | 9092 | 消息队列 |

---


## 第 25 章：故障排查手册

### 问题 1：Docker 启动不了

```bash
docker info
# 如果报错，Docker Desktop 没启动。打开 Docker Desktop，等鲸鱼图标稳定。
```

### 问题 2：`go run` 报 "cannot find package"

```bash
cd ~/Desktop/BlockExplore
pwd  # 确认在项目根目录
go mod tidy
```

### 问题 3：数据库连接失败 `dial tcp: lookup postgres: no such host`

**原因**：默认值用了 Docker 地址 `postgres`，但本地开发需要 `localhost`。

**解决**：确保 `.env` 中有 `DB_HOST=localhost`。看启动日志的 `[配置]` 行确认。

### 问题 4：Kafka 连不上 `lookup kafka: no such host`

**原因**：`.env` 中 `KAFKA_BROKERS` 还是 Docker 默认的 `kafka:9092`。

**解决**：`.env` 改为 `KAFKA_BROKERS=localhost:9092`。

### 问题 5：Kafka 报 "Unknown Topic Or Partition"

**原因**：Kafka 没开启自动建 topic，sync-worker 第一次发消息时 topic 不存在。

**解决**：`docker-compose.dev.yaml` 和 `docker-compose.yaml` 里 Kafka 加 `KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"`（本指南的配置已包含）。

### 问题 6：代理不起作用

日志显示"未使用代理"，但 `.env` 有 `HTTP_PROXY`。

**排查**：
1. 看启动日志的 `[配置]` 行，确认 viper 读到了代理
2. `config.go` 的 `Load()` 有 `os.Setenv("HTTP_PROXY", ...)`，确认执行了
3. 客户端用 `os.Getenv("HTTP_PROXY")` 读取
4. 确认代理软件在 7890 端口监听：`Test-NetConnection 127.0.0.1 -Port 7890`

### 问题 7：`npm run dev` 报端口占用 `EADDRINUSE :::3000`

```bash
netstat -ano | findstr :3000
taskkill /PID 找到的PID /F
```

### 问题 8：Docker 构建报 `go mod download` 失败（proxyconnect refused）

**原因**：Docker Desktop 全局配了代理 7890，但代理没开。

**解决**：打开代理 7890。Dockerfile 已设 `GOPROXY=goproxy.cn` 并清除代理 env 作为兜底。

### 问题 9：5432 端口被占用（Windows Hyper-V 保留）

```bash
netsh interface ipv4 show excludedportrange protocol=tcp
# 如果 5432 在保留段内
POSTGRES_PORT=5280 docker compose up -d
```

### 问题 10：访问 `/api/v1/price/*` 让 query-api 崩溃（panic）

**原因**：`cmd/query-api/main.go` 把 `priceHandler` 传成了 `nil`。

**解决**：按本指南第 20.1 节，创建真实的 `PriceService` 传给 `NewPriceHandler`。本指南版本已修复。

### 问题 11：集成测试报 `invalid input syntax for type numeric: ""`

**原因**：`Transaction.Value` 列是 `numeric(78,18)`，空字符串 `""` 无法插入。

**解决**：所有交易写入前给 `Value` 赋值（如 `"0"`）。client 代码已处理（`weiToEthStr` 永远返回带小数点的字符串）。

### 问题 12：重启后 Docker 数据丢失

`docker compose down` 不删数据卷。`down -v` 才删。开发阶段清空重建：
```bash
docker compose -f docker-compose.dev.yaml down -v
docker compose -f docker-compose.dev.yaml up -d
```

---

## 第 26 章：Git 分支管理与 Conventional Commits

> **为什么需要？** 专业项目和"玩具项目"的区别之一就是版本管理。规范的分支策略让你随时回退、并行尝试、清晰记录每次变更原因。

### 26.1 第一次提交

```bash
cd ~/Desktop/BlockExplore
git status  # 检查有没有不该提交的文件（.env 应该被忽略）
git add .
git commit -m "chore: init project scaffold

- Go microservices: query-api, search-api, price-api, sync workers, block-processor
- Kafka message queue (producer/consumer) with auto-create topics
- PostgreSQL + Redis infrastructure
- Next.js frontend with App Router
- Docker Compose dev/prod dual config
- Unit tests with testify (errcode/model/middleware/mq/service/handler)
- Integration tests (build tag) for repository and kafka
- Benchmarks for query service cache hit/miss"
```

### 26.2 Conventional Commits —— 提交信息规范

**格式**：`<type>(<scope>): <subject>`

```bash
# 好例子
git commit -m "feat(sync): add Solana slot tracking"
git commit -m "fix(query-api): wire real PriceService to avoid nil panic"
git commit -m "perf(query): add Redis cache for block detail (60s TTL)"
git commit -m "test(handler): add block list endpoint tests"

# 坏例子（避免）
git commit -m "update"
git commit -m "fix bug"
git commit -m "wip"
```

| type | 含义 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复 bug |
| `perf` | 性能优化 |
| `refactor` | 重构（不改变功能） |
| `test` | 添加/修改测试 |
| `docs` | 文档 |
| `chore` | 构建/工具/依赖 |
| `ci` | CI/CD 配置 |

### 26.3 Trunk-Based 分支策略

个人项目或小团队推荐 **Trunk-Based Flow**（比 GitFlow 轻量）：

```
main (生产就绪)
  └── feat/xxx (功能分支，1-3 天生命周期)
  └── fix/xxx  (修复分支，几小时)
  └── perf/xxx (性能优化分支)
```

**分支命名**：`feat/add-price-chart`、`fix/kafka-reconnect`、`test/add-benchmarks`。

### 26.4 实操：创建功能分支

```bash
# 从 main 切出新分支
git checkout -b feat/add-search-highlight

# 做修改... 编辑文件 ...
git diff  # 查看改动
git add internal/handler/search_handler.go
git commit -m "feat(search): add keyword highlighting"

# 切回 main 合并
git checkout main
git merge feat/add-search-highlight

# 删除已合并的分支
git branch -d feat/add-search-highlight
```

### 26.5 查看历史

```bash
git log --oneline --graph --all  # 简洁图形化历史
git log --stat                    # 带文件统计
git log -p internal/handler/block_handler.go  # 某文件的修改历史
```

---

## 第 27 章：Go 单元测试 —— 用代码验证代码

> **核心理念**：单元测试不是为了"找 bug"，而是为了让你**放心重构**。改了代码，跑一遍测试，全绿就说明没改坏。

### 27.1 Go 测试基础

```bash
go test ./...                  # 跑所有测试
go test -v ./...               # 带详细输出
go test -v ./internal/handler/ # 跑某个包
go test -v -run TestBlockHandler ./internal/handler/  # 跑特定函数
```

**测试文件命名**：`xxx_test.go`，放在和被测代码同一个包内。

### 27.2 errcode 测试（表驱动）

```bash
touch pkg/errcode/errcode_test.go
```

写入 `pkg/errcode/errcode_test.go`（完整代码对照仓库）：

```go
package errcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMsg(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"成功", CodeSuccess, "success"},
		{"参数错误", CodeBadRequest, "请求参数错误"},
		{"未找到", CodeNotFound, "资源不存在"},
		{"内部错误", CodeInternalError, "服务器内部错误"},
		{"数据库错误", CodeDBError, "数据库错误"},
		{"缓存错误", CodeCacheError, "缓存错误"},
		{"RPC 错误", CodeRPCError, "区块链 RPC 调用错误"},
		{"Kafka 错误", CodeKafkaError, "消息队列错误"},
		{"未知错误码", 999, "未知错误"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetMsg(tt.code))
		})
	}
}

func TestSuccess(t *testing.T) {
	data := map[string]int{"count": 42}
	resp := Success(data, "req-123")

	assert.NotNil(t, resp)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, data, resp.Data)
}

func TestError(t *testing.T) {
	resp := Error(CodeNotFound, "req-456")

	assert.NotNil(t, resp)
	assert.Equal(t, CodeNotFound, resp.Code)
	assert.Equal(t, "资源不存在", resp.Message)
	assert.Equal(t, "req-456", resp.RequestID)
	assert.Nil(t, resp.Data)
}

func TestErrorWithMsg(t *testing.T) {
	resp := ErrorWithMsg(CodeBadRequest, "chain 参数非法", "req-789")

	assert.NotNil(t, resp)
	assert.Equal(t, CodeBadRequest, resp.Code)
	assert.Equal(t, "chain 参数非法", resp.Message)
	assert.Equal(t, "req-789", resp.RequestID)
}
```

跑一下：

```bash
go test -v ./pkg/errcode/
```

**预期**：所有测试 PASS，覆盖率 100%。

### 27.3 model 测试

```bash
touch internal/model/model_test.go
```

写入 `internal/model/model_test.go`（完整代码对照仓库）。测试四个 `TableName()` 方法和字段赋值。

### 27.4 middleware 测试（用 httptest）

```bash
touch internal/middleware/middleware_test.go
```

写入 `internal/middleware/middleware_test.go`（完整代码对照仓库）。测试：
- `RequestID` 生成新 ID / 复用入站 header
- `CORS` 设置响应头 / 处理 OPTIONS 预检
- `RateLimiter` 突发容量 / IP 隔离 / 拒绝并返回 429

### 27.5 service 层测试（接口 + 手写 mock）

> **核心思想**：`QueryService` 依赖接口（`BlockRepository`/`TxRepository`/`Cacher`），测试时传入手写 mock 实现，不连真实数据库。

```bash
touch internal/service/query/query_service_test.go
```

写入 `internal/service/query/query_service_test.go`（完整代码对照仓库）。测试：
- `GetBlockList` 缓存未命中 → 查 DB → 写缓存
- `GetBlockList` 缓存命中 → 跳过 DB
- `GetBlockList` DB 错误
- `GetBlockList` nil cache 不 panic
- `GetBlockTransactions` / `GetTransactionDetail` / `GetAddressTransactions`

### 27.6 processor 测试

```bash
touch internal/service/processor/block_processor_test.go
```

写入 `internal/service/processor/block_processor_test.go`（完整代码对照仓库）。测试：
- `Handle` 成功（区块+交易都写入，交易 BlockID 被正确设置）
- `Handle` 无交易（不调用 tx 写入）
- `Handle` 区块写入失败（不继续写交易）
- `Handle` 交易写入失败
- `Handle` 无效数据

### 27.7 handler 测试（httptest + mock service）

```bash
touch internal/handler/handler_test.go
```

写入 `internal/handler/handler_test.go`（完整代码对照仓库）。测试：
- `GetBlockList` 成功（200）
- `GetBlockList` 默认参数
- `GetBlockDetail` 成功 / 无效数字（400）/ 未找到（404）
- `GetBlockTransactions` 成功

### 27.8 测试运行命令速查

```bash
go test ./...                           # 跑所有单元测试
go test -v ./...                        # 详细输出
go test -v -run TestBlockHandler ./internal/handler/  # 特定函数
go test -failfast ./...                 # 失败时停止
go test -cover ./...                    # 显示覆盖率
go test -count=5 ./internal/handler/    # 跑 5 次（排查偶发失败）
go test -timeout 30s ./...              # 超时设置
```

---

## 第 28 章：集成测试 —— 端到端验证

> **单元测试验证"零件"正确，集成测试验证"组装"正确。**

集成测试用 `//go:build integration` 标签隔离，默认 `go test ./...` 不跑它们，需要显式 `-tags=integration`。

### 28.1 repository 集成测试（真实 PostgreSQL）

```bash
touch internal/repository/repository_integration_test.go
```

写入 `internal/repository/repository_integration_test.go`（完整代码对照仓库）。文件顶部必须有：

```go
//go:build integration

package repository
```

使用 `testify/suite` 组织测试套件：
- `SetupSuite`：连接测试数据库 `blockexplore_test`，`AutoMigrate` 建表
- `TearDownSuite`：删表
- `SetupTest`：每个测试前清空数据

测试：`TestCreateSingle`、`TestGetByChainAndNumber`、`TestGetLatest`、`TestGetList_Pagination`、`TestTxRepo_GetByAddress`。

### 28.2 运行集成测试前准备

```bash
# 1. 启动基础设施
docker compose -f docker-compose.dev.yaml up -d

# 2. 创建测试数据库
docker exec blockexplore-dev-postgres psql -U blockexplore -c "CREATE DATABASE blockexplore_test;"

# 3. 运行（如果用了备用端口）
DB_PORT=5280 go test -v -tags=integration ./internal/repository/
```

### 28.3 Kafka 集成测试

```bash
touch internal/mq/kafka_integration_test.go
```

写入 `internal/mq/kafka_integration_test.go`（完整代码对照仓库）。文件顶部：

```go
//go:build integration

package mq
```

测试 `TestKafka_ProduceAndConsume`：发送一条 `BlockMessage` → 消费 → 验证内容。带 15 秒超时。

```bash
go test -v -tags=integration -run TestKafka ./internal/mq/
```

### 28.4 区分运行

```bash
go test ./...                              # 只跑单元测试（默认）
go test -tags=integration ./...            # 单元 + 集成测试
go test -short ./...                       # 跳过集成测试（测试代码里用 testing.Short() 判断）
```

---

## 第 29 章：性能基准测试 —— benchmark

> **"Redis 缓存提高了查询性能"——提高了多少？Benchmark 用来回答这个问题。**

### 29.1 写 benchmark

```bash
touch internal/service/query/query_benchmark_test.go
```

写入 `internal/service/query/query_benchmark_test.go`（完整代码对照仓库）：

```go
package query

import (
	"context"
	"testing"

	"blockexplore/internal/model"
)

// BenchmarkGetBlockList_CacheMiss 模拟缓存未命中（每次都查 DB mock）
func BenchmarkGetBlockList_CacheMiss(b *testing.B) {
	repo := &mockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		total:  1,
	}
	svc := NewQueryService(repo, &mockTxRepo{}, nil) // nil cache = 永远未命中
	ctx := context.Background()
	_ = ctx

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetBlockList("eth", 1, 20)
	}
}

// BenchmarkGetBlockList_CacheHit 模拟缓存命中（不查 DB）
func BenchmarkGetBlockList_CacheHit(b *testing.B) {
	repo := &mockBlockRepo{}
	cache := newMockCache()
	preloaded := &BlockListResponse{
		Chain:  "eth",
		Blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		Pagination: Pagination{Page: 1, PageSize: 20, Total: 1},
	}
	_ = cache.Set(context.Background(), "blocks:eth:1:20", preloaded, 0)

	svc := NewQueryService(repo, &mockTxRepo{}, cache)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetBlockList("eth", 1, 20)
	}
}

// BenchmarkGetBlockList_Parallel 并发场景（nil cache 避免 mock map 并发写）
func BenchmarkGetBlockList_Parallel(b *testing.B) {
	repo := &mockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		total:  1,
	}
	svc := NewQueryService(repo, &mockTxRepo{}, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = svc.GetBlockList("eth", 1, 20)
		}
	})
}
```

### 29.2 运行 benchmark

> **PowerShell 注意**：`-bench=.` 里的 `.` 会被 PowerShell 当成包路径。**必须加引号**：`"-bench=."`

```bash
go test "-bench=." -benchmem -run=XXX_NONE ./internal/service/query/
```

**输出示例**：

```
goos: windows
goarch: amd64
pkg: blockexplore/internal/service/query
cpu: AMD Ryzen 7 8845H
BenchmarkGetBlockList_CacheMiss-16     10567938    117.5 ns/op    96 B/op    3 allocs/op
BenchmarkGetBlockList_CacheHit-16        415084   2804 ns/op    640 B/op   14 allocs/op
BenchmarkGetBlockList_Parallel-16      22306663     51.42 ns/op   96 B/op   3 allocs/op
PASS
```

**输出格式解读**：

| 列 | 含义 |
|----|------|
| `BenchmarkGetBlockList_CacheMiss-16` | 名称，`-16` = 用了 16 个 CPU 核心 |
| `10567938` | 运行次数（越大越好） |
| `117.5 ns/op` | 每次操作耗时 117.5 纳秒（越小越好） |
| `96 B/op` | 每次操作分配 96 字节内存 |
| `3 allocs/op` | 每次操作 3 次内存分配 |

> **注意**：上面的 `CacheHit` 比 `CacheMiss` 慢，是因为 mock cache 用了 `encoding/json` 序列化/反序列化，比 mock repo 直接返回切片还慢。真实 Redis（内存 KV）比真实 PostgreSQL（磁盘 + SQL 解析）快得多。要测真实缓存收益，需用真实 Redis（见第 28 章集成测试思路）。

### 29.3 常用参数

```bash
go test "-bench=." -benchtime=3s ./...               # 每个 bench 至少跑 3 秒
go test "-bench=." -count=10 ./internal/service/query/ | tee bench_results.txt  # 跑 10 次取平均
```

---

## 第 30 章：测试覆盖率

> **覆盖率不是越高越好，但它告诉你哪些代码完全没被测试过——那些是潜在的炸弹。**

### 30.1 生成覆盖率报告

```bash
go test -coverprofile=coverage.out -run=XXX_NONE ./...
go tool cover -func=coverage.out
```

**输出示例**：
```
blockexplore/pkg/errcode/errcode.go:12:    Success     100.0%
blockexplore/internal/model/block.go:35:   TableName    100.0%
blockexplore/internal/handler/block.go:50: GetBlockList 85.7%
total:                                    (statements) 42.3%
```

### 30.2 HTML 可视化

```bash
go tool cover -html=coverage.out -o coverage.html
start coverage.html  # Windows 用默认浏览器打开
```

绿色 = 被覆盖，红色 = 没被执行过，灰色 = 非可执行行。

### 30.3 覆盖率脚本

```bash
touch scripts/check_coverage.sh
```

写入 `scripts/check_coverage.sh`：

```bash
#!/bin/bash
# 检查测试覆盖率是否达标
# 用法: bash scripts/check_coverage.sh [阈值，默认 40]

set -e

THRESHOLD=${1:-40}

echo "==> 运行测试并生成覆盖率报告..."
go test -coverprofile=coverage.out -covermode=atomic -run=XXX_NONE ./... 2>&1

# 提取总覆盖率（百分比数字）
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "=========================================="
echo "总测试覆盖率: ${COVERAGE}%"
echo "最低要求:     ${THRESHOLD}%"
echo "=========================================="

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "FAIL: 测试覆盖率不达标！"
    exit 1
else
    echo "OK: 测试覆盖率达标。"
    exit 0
fi
```

```bash
bash scripts/check_coverage.sh 40
```

### 30.4 覆盖率策略

| 层级 | 建议覆盖率 | 原因 |
|------|----------|------|
| `pkg/errcode` | 100% | 简单工具，应全量覆盖 |
| `pkg/logger` | 60% | 初始化逻辑难以测试 |
| `internal/model` | 80% | 重点测 TableName 和字段标签 |
| `internal/repository` | 70% | 纯 SQL 操作，集成测试覆盖 |
| `internal/service` | 80% | 业务逻辑核心，必须重点覆盖 |
| `internal/handler` | 75% | 重点测参数校验/错误码 |
| `internal/mq` | 50% | 集成测试成本高，核心流程覆盖即可 |

---

## 第 31 章：CI/CD —— GitHub Actions 自动化流水线

> **CI/CD 的意义**：每次 push 自动跑测试，在合并前发现问题，而不是部署后才发现。

### 31.1 创建工作流

```bash
mkdir -p .github/workflows
touch .github/workflows/ci.yml
```

写入 `.github/workflows/ci.yml`（完整代码对照仓库）。四阶段流水线：

1. **lint**：golangci-lint + go vet
2. **test**：单元测试 + 覆盖率（`-run=XXX_NONE` 排除集成测试）
3. **build**：编译全部 7 个服务
4. **integration**：启动 PostgreSQL + Redis 服务容器，跑 `-tags=integration` 测试

```yaml
name: CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  lint:
    name: Lint & Vet
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m
      - name: go vet
        run: go vet ./...

  test:
    name: Unit Tests
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      - name: Run unit tests (exclude integration)
        run: go test -v -coverprofile=coverage.out -covermode=atomic -run=XXX_NONE ./...
      - name: Show coverage
        run: |
          go tool cover -func=coverage.out | tail -1
      - name: Upload coverage artifact
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      - name: Build all 7 services
        run: |
          mkdir -p bin
          go build -ldflags="-s -w" -o bin/query-api       ./cmd/query-api/
          go build -ldflags="-s -w" -o bin/search-api      ./cmd/search-api/
          go build -ldflags="-s -w" -o bin/price-api       ./cmd/price-api/
          go build -ldflags="-s -w" -o bin/eth-sync-worker ./cmd/eth-sync-worker/
          go build -ldflags="-s -w" -o bin/btc-sync-worker ./cmd/btc-sync-worker/
          go build -ldflags="-s -w" -o bin/sol-sync-worker ./cmd/sol-sync-worker/
          go build -ldflags="-s -w" -o bin/block-processor ./cmd/block-processor/
      - name: Upload binaries
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: bin/

  integration:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: build
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: blockexplore
          POSTGRES_PASSWORD: blockexplore123
          POSTGRES_DB: blockexplore_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      - name: Run integration tests
        env:
          DB_HOST: localhost
          DB_PORT: 5432
        run: go test -v -tags=integration ./internal/repository/
```

### 31.2 PR 模板

```bash
touch .github/pull_request_template.md
```

写入 `.github/pull_request_template.md`（完整内容对照仓库）。

### 31.3 提交 CI 配置

```bash
git add .github/
git commit -m "ci: add GitHub Actions workflow (lint, test, build, integration)"
```

### 31.4 CI 流水线流程

```
你 push 代码
    │
    ▼
┌─────────────┐
│  Job 1: Lint │  ← golangci-lint + go vet
└──────┬──────┘     失败 → 通知你修复
       │ 通过
       ▼
┌─────────────┐
│  Job 2: Test │  ← 单元测试 + 覆盖率
└──────┬──────┘     失败 → 通知你修复
       │ 通过
       ▼
┌─────────────┐
│ Job 3: Build │  ← 编译 7 个微服务
└──────┬──────┘     失败 → 编译错误
       │ 通过
       ▼
┌──────────────┐
│ Job 4:       │  ← 启动 PostgreSQL + Redis
│ Integration  │     跑集成测试
└──────┬───────┘
       │ 全部通过
       ▼
   ✅ 可以合并到 main
```

---

## 第 32 章：代码质量工具

### 32.1 golangci-lint 配置

```bash
touch .golangci.yml
```

写入 `.golangci.yml`（完整代码对照仓库）。启用的 linter：`errcheck`、`gosimple`、`govet`、`ineffassign`、`staticcheck`、`unused`、`gofmt`、`goimports`、`misspell`、`bodyclose`。

```bash
# 安装
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行
golangci-lint run ./...

# 只检查新代码
golangci-lint run --new-from-rev=main ./...
```

### 32.2 go vet

```bash
go vet ./...
```

### 32.3 格式化

```bash
gofmt -w ./internal ./pkg ./cmd
go install golang.org/x/tools/cmd/goimports@latest
goimports -w ./internal ./pkg ./cmd
```

### 32.4 Pre-commit Hook

```bash
touch scripts/hooks/pre-commit
```

写入 `scripts/hooks/pre-commit`（完整代码对照仓库）：

```bash
#!/bin/bash
# Git pre-commit hook: 提交前自动检查代码格式和 vet
# 安装方式: git config core.hooksPath scripts/hooks

echo "==> 运行提交前检查..."

# 1. 检查 Go 代码格式
UNFORMATTED=$(gofmt -l ./internal ./pkg ./cmd 2>/dev/null)
if [ -n "$UNFORMATTED" ]; then
    echo "FAIL: 以下文件格式不正确:"
    echo "$UNFORMATTED"
    echo "运行 'gofmt -w ./internal ./pkg ./cmd' 修复"
    exit 1
fi

# 2. 快速 vet
go vet ./... 2>&1 | grep -v "no test files" || true

# 3. 快速单元测试（失败则中断提交）
if ! go test -run=XXX_NONE ./... >/dev/null 2>&1; then
    echo "FAIL: 单元测试未通过，请运行 'go test ./...' 查看详情"
    exit 1
fi

echo "OK: 提交前检查通过。"
```

启用：

```bash
git config core.hooksPath scripts/hooks
```

---

## 第 33 章：对外部署 —— 让全世界访问你的浏览器

到目前为止，你的浏览器跑在 `localhost:3000`，只有你自己能访问。这一章讲三种"让外人也能访问"的方式，从易到难。

### 33.1 方式一：内网穿透（最快，适合临时演示）

用 [cloudflared](https://github.com/cloudflare/cloudflared) 或 [ngrok](https://ngrok.com/) 把本地端口暴露到公网。

**cloudflared（免费，无需注册）**：

```bash
# 下载 cloudflared
winget install --id Cloudflare.cloudflared

# 暴露前端 3000 端口
cloudflared tunnel --url http://localhost:3000
```

你会看到：
```
Your quick Tunnel has been created! Visit it at:
  https://xxx-yyy-zzz.trycloudflare.com
```

把这个网址发给别人，他们就能访问你的浏览器了。

> **注意**：内网穿透把流量转到你本机的 3000。确保 `docker compose up` 正在运行，且代理 7890 已开（sync-worker 需要拉链上数据）。cloudflared 免费隧道有访问频次限制，适合临时演示，不适合长期运行。

### 33.2 方式二：云服务器部署（适合长期运行）

**步骤**：

1. **买一台云服务器**（阿里云/腾讯云/AWS/Vultr），推荐 Ubuntu 22.04，2 核 4G 起步。

2. **安装 Docker**：
   ```bash
   ssh your-user@your-server-ip
   sudo apt update && sudo apt install -y docker.io docker-compose-plugin
   sudo usermod -aG docker $USER
   # 重新登录使组权限生效
   ```

3. **把代码传上去**（任选一种）：
   - 推到 GitHub，在服务器 `git clone`
   - 或用 `scp -r ~/Desktop/BlockExplore your-user@your-server-ip:~/`

4. **配置 `.env.docker`**：
   ```bash
   cd BlockExplore
   # 如果服务器在国内，保留代理设置（需要服务器上也有代理）
   # 如果服务器在海外，删掉 HTTP_PROXY/HTTPS_PROXY 两行（或留空）
   ```

5. **构建并启动**：
   ```bash
   docker compose up -d --build
   docker compose ps  # 确认全部 Up
   ```

6. **配置防火墙开放端口**：
   ```bash
   sudo ufw allow 3000  # 前端
   sudo ufw allow 8080  # API（可选，如果想让外人直接调 API）
   ```

7. **访问**：`http://your-server-ip:3000`

8. **绑定域名 + HTTPS**（推荐用 Caddy，自动签证书）：
   ```bash
   sudo apt install -y caddy
   sudo systemctl enable caddy
   ```
   编辑 `/etc/caddy/Caddyfile`：
   ```
   blockexplore.yourdomain.com {
       reverse_proxy localhost:3000
   }
   ```
   ```bash
   sudo systemctl restart caddy
   ```
   Caddy 会自动申请 Let's Encrypt 证书。访问 `https://blockexplore.yourdomain.com`。

### 33.3 方式三：Docker 镜像推到 Registry + 服务器拉取（最专业）

适合多次部署、CI/CD 自动部署。

1. **在 Docker Hub 或阿里云 ACR 创建仓库**。

2. **本地构建并推送**：
   ```bash
   docker login
   docker build -t yourusername/blockexplore-query-api:latest .
   docker push yourusername/blockexplore-query-api:latest
   # 同样推送 web 镜像
   cd web && docker build -t yourusername/blockexplore-web:latest . && docker push yourusername/blockexplore-web:latest
   ```

3. **在 GitHub Actions 里加 CD job**（可选）：构建后自动推送，触发服务器更新。

4. **服务器拉取运行**：
   ```bash
   docker pull yourusername/blockexplore-query-api:latest
   # 修改 docker-compose.yaml 用 image: 而不是 build:
   docker compose up -d
   ```

### 33.4 部署后验证

无论哪种方式，部署后都验证：
```bash
curl https://你的域名/health          # 应返回 {"status":"ok"}
curl https://你的域名/api/v1/blocks?chain=eth  # 应返回区块数据
```

打开浏览器访问 `https://你的域名`，确认能看到价格卡片和区块列表。

---

## 总结：从零到一的完整路径

回顾你从空文件夹到现在拥有的一切：

```
第 0 章：检查环境 ✓
第 1 章：创建目录骨架 + Git 初始化 + .gitignore ✓
第 2 章：Docker 启动 PostgreSQL + Redis + Kafka（含 auto-create topics）✓
第 3 章：建表 SQL（4 张表 + 索引）✓
第 4 章：Go module 初始化 + 9 个依赖（含 testify）✓
第 5 章：config 包（viper，mapstructure tag）✓
第 6 章：logger 包（zap）✓
第 7 章：cache 包（RedisClient 结构体 + 8 个方法）✓
第 8 章：errcode 包（8 个错误码，返回 *Response）✓
第 9 章：model 包（4 个模型：Block/Transaction/Address/PriceHistory）✓
第 10 章：repository 包（BlockRepo/TxRepo/SearchRepo/PriceRepo）✓
第 11 章：mq 包（Kafka Producer/Consumer + ConsumeAll）✓
第 12 章：client 包（ETH JSON-RPC / BTC BlockCypher / SOL JSON-RPC）✓
第 13 章：sync 包（3 个 Worker，SOL 带重试退避）✓
第 14 章：processor 包（接口依赖，可 mock）✓
第 15 章：price 包（CoinGecko + cron 定时同步 + 缓存）✓
第 16 章：query 包（Cache-Aside，接口依赖，可 mock）✓
第 17 章：handler 包（4 个 Handler）✓
第 18 章：middleware 包（CORS/RequestID/RateLimiter 令牌桶）✓
第 19 章：router 包（统一路由注册）✓
第 20 章：cmd/（7 个微服务，query-api 修复了 nil panic）✓
第 21 章：编译验证，go run 跑通 ✓
第 22 章：Next.js 前端（App Router + 6 组件 + 5 页面）✓
第 23 章：Docker 容器化（多阶段构建 + GOPROXY + 11 服务 compose）✓
第 24 章：完整运行与端到端验证（真实链上数据入库）✓
第 25 章：故障排查手册（12 个常见问题）✓
第 26 章：Git 分支管理与 Conventional Commits ✓
第 27 章：Go 单元测试（testify + 表驱动 + 手写 mock + httptest）✓
第 28 章：集成测试（build tag 隔离，真实 PG + Kafka）✓
第 29 章：性能基准测试（benchmark，缓存命中/未命中对比）✓
第 30 章：测试覆盖率（HTML 报告 + 阈值脚本）✓
第 31 章：CI/CD（GitHub Actions 四阶段：lint→test→build→integration）✓
第 32 章：代码质量工具（golangci-lint + gofmt + pre-commit hook）✓
第 33 章：对外部署（cloudflared 内网穿透 / 云服务器 / Docker Registry）✓
```

**一条数据的完整旅程**：

```
区块链节点
  ↓ RPC/API 拉取
eth-sync-worker (Go)
  ↓ Kafka Producer
Kafka Topic "block.raw.eth"
  ↓ Kafka Consumer
block-processor (Go)
  ↓ GORM INSERT
PostgreSQL (blocks + transactions 表)
  ↑ GORM SELECT
query-api (Go) ← Redis 缓存
  ↓ HTTP JSON
Next.js (rewrites 代理)
  ↓ React 渲染
浏览器 (Etherscan 风格深色界面)
```

**你现在拥有了**：
- 一个完整的 Go 微服务项目（7 个独立服务，能编译能跑）
- Kafka 异步消息队列的生产/消费链路（auto-create topics）
- Redis 缓存加速查询（Cache-Aside 模式，`RedisClient` 结构体封装）
- 三层架构（Handler → Service → Repository），service/processor 用接口定义依赖，可 mock 测试
- 多链数据归一化适配（ETH 18 位 / BTC 8 位 / SOL 9 位）
- Next.js 14 前端（App Router + Tailwind 深色主题）
- Docker 容器化（多阶段构建，GOPROXY 加速，11 服务 compose）
- 开发/生产双环境配置（`.env` + `.env.docker`）
- 完整的测试体系：单元测试（testify）+ 集成测试（build tag）+ Benchmark
- CI/CD 自动化流水线（GitHub Actions：lint → test → build → integration）
- Git 分支管理规范（Conventional Commits + Trunk-Based Flow）
- 代码质量工具（golangci-lint + pre-commit hook）
- 三种对外部署方式（内网穿透 / 云服务器 / Docker Registry）
- **修复了原 demo 的关键 bug**：query-api 的 nil panic、Kafka topic 不自动创建、Docker 构建代理问题

**本指南与项目代码逐字对应**。把项目全删了，照着这份指南从头到尾做一遍，你能 100% 复现这个能编译、能跑测试、有 CI、能 Docker 部署、对外可访问的多链区块链浏览器。

---

> 指南到此结束。遇到问题先看第 25 章故障排查，再看对应章节的"编译检查/验证"步骤。祝构建顺利。

