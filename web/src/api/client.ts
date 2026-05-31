// ============================================================
// API 客户端
// 根据后端实际 API 响应格式封装调用方法
// ============================================================
import axios from 'axios';
import type {
  ApiResponse,
  BlockListResponse,
  TxListResponse,
  Block,
  Transaction,
  PriceResponse,
  PriceHistoryResponse,
  SearchResult,
  ChainType,
} from '../types';

// 创建 axios 实例
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

// ============================================================
// 区块相关 API
// ============================================================

// 获取区块列表
export async function getBlockList(
  chain: ChainType,
  page: number = 1,
  pageSize: number = 20
): Promise<BlockListResponse> {
  const resp = await api.get<ApiResponse<BlockListResponse>>('/blocks', {
    params: { chain, page, page_size: pageSize },
  });
  return resp.data.data;
}

// 获取区块详情
export async function getBlockDetail(
  chain: ChainType,
  blockNumber: number
): Promise<Block> {
  const resp = await api.get<ApiResponse<Block>>(`/blocks/${blockNumber}`, {
    params: { chain },
  });
  return resp.data.data;
}

// 获取区块内的交易列表
export async function getBlockTransactions(
  chain: ChainType,
  blockNumber: number,
  page: number = 1,
  pageSize: number = 20
): Promise<TxListResponse> {
  const resp = await api.get<ApiResponse<TxListResponse>>(
    `/blocks/${blockNumber}/transactions`,
    { params: { chain, page, page_size: pageSize } }
  );
  return resp.data.data;
}

// ============================================================
// 交易相关 API
// ============================================================

// 获取交易详情
export async function getTransactionDetail(
  chain: ChainType,
  txHash: string
): Promise<Transaction> {
  const resp = await api.get<ApiResponse<Transaction>>(`/transactions/${txHash}`, {
    params: { chain },
  });
  return resp.data.data;
}

// 获取地址的交易历史
export async function getAddressTransactions(
  chain: ChainType,
  address: string,
  page: number = 1,
  pageSize: number = 20
): Promise<TxListResponse> {
  const resp = await api.get<ApiResponse<TxListResponse>>(
    `/addresses/${address}/transactions`,
    { params: { chain, page, page_size: pageSize } }
  );
  return resp.data.data;
}

// ============================================================
// 搜索相关 API
// ============================================================

export async function search(keyword: string): Promise<SearchResult> {
  const resp = await api.get<ApiResponse<SearchResult>>('/search', {
    params: { q: keyword },
  });
  return resp.data.data;
}

// ============================================================
// 价格相关 API
// ============================================================

// 获取当前价格
export async function getCurrentPrice(chain: ChainType): Promise<PriceResponse> {
  const resp = await api.get<ApiResponse<PriceResponse>>(`/price/${chain}`);
  return resp.data.data;
}

// 获取价格历史
export async function getPriceHistory(
  chain: ChainType,
  limit: number = 100
): Promise<PriceHistoryResponse> {
  const resp = await api.get<ApiResponse<PriceHistoryResponse>>(`/price/${chain}/history`, {
    params: { limit },
  });
  return resp.data.data;
}
