const API_BASE = '/api/v1'

async function fetchAPI<T>(path: string, params?: Record<string, string | number>): Promise<T> {
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.set(key, String(value))
    })
  }
  const resp = await fetch(url.toString())
  if (!resp.ok) throw new Error(`API error: ${resp.status}`)
  const json = await resp.json()
  return json.data
}

export interface Block {
  id: number
  chain: string
  block_number: number
  block_hash: string
  parent_hash: string
  timestamp: number
  tx_count: number
  gas_used: string
  gas_limit: string
  difficulty: string
  size_bytes: number
  slot: number | null
  created_at: string
}

export interface Transaction {
  id: number
  chain: string
  tx_hash: string
  block_number: number
  from_addr: string
  to_addr: string
  value: string
  gas_price: string
  gas_used: string
  gas_limit: string
  nonce: number
  input_data: string
  status: number
  timestamp: number
  created_at: string
}

export interface PriceResponse {
  chain: string
  symbol: string
  price_usd: number
  timestamp: number
}

export interface PriceHistoryItem {
  id: number
  chain: string
  symbol: string
  price_usd: string
  timestamp: number
  created_at: string
}

export interface Pagination {
  page: number
  page_size: number
  total: number
}

export interface BlockListResponse {
  chain: string
  blocks: Block[]
  pagination: Pagination
}

export interface TxListResponse {
  chain: string
  transactions: Transaction[]
  pagination: Pagination
}

export interface PriceHistoryResponse {
  chain: string
  symbol: string
  prices: PriceHistoryItem[]
}

export type ChainType = 'eth' | 'btc' | 'sol'

export const CHAIN_CONFIG: Record<ChainType, { name: string; symbol: string; color: string; icon: string }> = {
  eth: { name: 'Ethereum', symbol: 'ETH', color: '#627eea', icon: 'Ξ' },
  btc: { name: 'Bitcoin', symbol: 'BTC', color: '#f7931a', icon: '₿' },
  sol: { name: 'Solana', symbol: 'SOL', color: '#9945ff', icon: '◎' },
}

export async function getBlockList(chain: ChainType, page = 1, pageSize = 20): Promise<BlockListResponse> {
  return fetchAPI('/blocks', { chain, page, page_size: pageSize })
}

export async function getBlockDetail(chain: ChainType, blockNumber: number): Promise<Block> {
  return fetchAPI(`/blocks/${blockNumber}`, { chain })
}

export async function getBlockTransactions(chain: ChainType, blockNumber: number, page = 1, pageSize = 20): Promise<TxListResponse> {
  return fetchAPI(`/blocks/${blockNumber}/transactions`, { chain, page, page_size: pageSize })
}

export async function getTransactionDetail(chain: ChainType, txHash: string): Promise<Transaction> {
  return fetchAPI(`/transactions/${txHash}`, { chain })
}

export async function getAddressTransactions(chain: ChainType, address: string, page = 1, pageSize = 20): Promise<TxListResponse> {
  return fetchAPI(`/addresses/${address}/transactions`, { chain, page, page_size: pageSize })
}

export async function getCurrentPrice(chain: ChainType): Promise<PriceResponse> {
  return fetchAPI(`/price/${chain}`)
}

export async function getPriceHistory(chain: ChainType, limit = 200): Promise<PriceHistoryResponse> {
  return fetchAPI(`/price/${chain}/history`, { limit })
}
