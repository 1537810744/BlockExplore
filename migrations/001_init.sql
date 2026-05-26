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
