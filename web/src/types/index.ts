// ============================================================
// 类型定义文件
// 根据后端实际 API 响应格式定义
// ============================================================

// 区块数据类型
export interface Block {
  id: number;
  chain: string;
  block_number: number;
  block_hash: string;
  parent_hash: string;
  timestamp: number;
  tx_count: number;
  miner: string;
  gas_used: string;
  gas_limit: string;
  difficulty: string;
  size_bytes: number;
  slot: number | null;
  created_at: string;
}

// 交易数据类型
export interface Transaction {
  id: number;
  chain: string;
  tx_hash: string;
  block_number: number;
  from_addr: string;
  to_addr: string;
  value: string;
  gas_price: string;
  gas_used: string;
  gas_limit: string;
  nonce: number;
  input_data: string;
  status: number;
  timestamp: number;
  created_at: string;
}

// 当前价格响应（GET /api/v1/price/:chain）
export interface PriceResponse {
  chain: string;
  symbol: string;
  price_usd: number;
  timestamp: number;
}

// 价格历史条目（GET /api/v1/price/:chain/history）
export interface PriceHistoryItem {
  id: number;
  chain: string;
  symbol: string;
  price_usd: string;
  timestamp: number;
  created_at: string;
}

// 价格历史响应
export interface PriceHistoryResponse {
  chain: string;
  symbol: string;
  prices: PriceHistoryItem[];
}

// 搜索结果类型
export interface SearchResult {
  type: 'block' | 'transaction' | 'address';
  data: Block | Transaction | AddressInfo;
}

// 地址信息类型
export interface AddressInfo {
  address: string;
  balance: string;
  tx_count: number;
}

// 分页信息
export interface Pagination {
  page: number;
  page_size: number;
  total: number;
}

// 区块列表响应（GET /api/v1/blocks）
export interface BlockListResponse {
  chain: string;
  blocks: Block[];
  pagination: Pagination;
}

// 交易列表响应（GET /api/v1/blocks/:number/transactions）
export interface TxListResponse {
  chain: string;
  transactions: Transaction[];
  pagination: Pagination;
}

// API 统一响应类型
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  request_id: string;
}

// 链类型
export type ChainType = 'eth' | 'btc' | 'sol';
