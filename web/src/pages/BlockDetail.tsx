// ============================================================
// BlockDetail 区块详情页面
// ============================================================
import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getBlockDetail, getBlockTransactions } from '../api/client'
import type { Block, Transaction, ChainType } from '../types'

export default function BlockDetail() {
  const { chain, blockNumber } = useParams<{ chain: string; blockNumber: string }>()
  const [block, setBlock] = useState<Block | null>(null)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const pageSize = 20

  useEffect(() => {
    const fetchData = async () => {
      if (!chain || !blockNumber) return
      setLoading(true)
      setError('')
      try {
        const blockNum = parseInt(blockNumber)
        const [blockData, txData] = await Promise.all([
          getBlockDetail(chain as ChainType, blockNum),
          getBlockTransactions(chain as ChainType, blockNum, page, pageSize),
        ])
        setBlock(blockData)
        setTransactions(txData.transactions || [])
        setTotal(txData.pagination?.total || 0)
      } catch {
        setError('获取区块详情失败')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [chain, blockNumber, page])

  const formatTime = (ts: number) => new Date(ts * 1000).toLocaleString('zh-CN')
  const shorten = (s: string, n = 10) =>
    s ? `${s.slice(0, n)}...${s.slice(-8)}` : '-'

  const totalPages = Math.ceil(total / pageSize)

  if (loading) {
    return <div className="flex justify-center items-center h-64 text-slate-400">加载中...</div>
  }
  if (error || !block) {
    return <div className="flex justify-center items-center h-64 text-red-400">{error || '区块不存在'}</div>
  }

  return (
    <div>
      <div className="mb-6">
        <Link to="/" className="text-blue-400 hover:text-blue-300 text-sm mb-2 inline-block">
          ← 返回区块列表
        </Link>
        <h1 className="text-2xl font-bold text-white">区块 #{block.block_number}</h1>
      </div>

      <div className="bg-slate-800 rounded-lg p-6 mb-6">
        <h2 className="text-lg font-semibold text-white mb-4">基本信息</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <DetailRow label="区块高度" value={block.block_number.toString()} />
          <DetailRow label="时间戳" value={formatTime(block.timestamp)} />
          <DetailRow label="交易数" value={`${block.tx_count} 笔`} />
          <DetailRow label="Gas 使用" value={block.gas_used || '-'} />
          <DetailRow label="Gas 上限" value={block.gas_limit || '-'} />
          <DetailRow label="区块大小" value={`${block.size_bytes} 字节`} />
        </div>
        <div className="mt-4 pt-4 border-t border-slate-700">
          <DetailRow label="区块哈希" value={block.block_hash} mono />
          <DetailRow label="父区块哈希" value={block.parent_hash} mono />
        </div>
      </div>

      <div className="bg-slate-800 rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-700 flex justify-between items-center">
          <h2 className="text-lg font-semibold text-white">
            交易列表 ({total} 笔)
          </h2>
          {totalPages > 1 && (
            <span className="text-slate-400 text-sm">第 {page}/{totalPages} 页</span>
          )}
        </div>
        {transactions.length === 0 ? (
          <div className="px-6 py-8 text-center text-slate-400">暂无交易</div>
        ) : (
          <>
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700 text-slate-400 text-sm">
                  <th className="px-6 py-3 text-left">交易哈希</th>
                  <th className="px-6 py-3 text-left">从</th>
                  <th className="px-6 py-3 text-left">到</th>
                  <th className="px-6 py-3 text-left">金额</th>
                  <th className="px-6 py-3 text-left">状态</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx) => (
                  <tr key={tx.tx_hash} className="border-b border-slate-700/50 hover:bg-slate-700/30">
                    <td className="px-6 py-4">
                      <Link to={`/tx/${chain}/${tx.tx_hash}`} className="text-blue-400 hover:text-blue-300 font-mono text-sm">
                        {shorten(tx.tx_hash)}
                      </Link>
                    </td>
                    <td className="px-6 py-4 font-mono text-sm text-slate-300">{shorten(tx.from_addr)}</td>
                    <td className="px-6 py-4 font-mono text-sm text-slate-300">
                      {tx.to_addr ? shorten(tx.to_addr) : '(合约创建)'}
                    </td>
                    <td className="px-6 py-4 text-sm text-slate-300">{tx.value || '0'}</td>
                    <td className="px-6 py-4">
                      <span className={`px-2 py-1 rounded text-xs ${
                        tx.status === 1 ? 'bg-green-900/50 text-green-400' : 'bg-red-900/50 text-red-400'
                      }`}>
                        {tx.status === 1 ? '成功' : '失败'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {totalPages > 1 && (
              <div className="flex justify-center items-center gap-4 py-4 border-t border-slate-700">
                <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}
                  className="px-4 py-2 bg-slate-700 rounded-lg text-sm hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed">
                  上一页
                </button>
                <span className="text-slate-400">第 {page} / {totalPages} 页</span>
                <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}
                  className="px-4 py-2 bg-slate-700 rounded-lg text-sm hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed">
                  下一页
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function DetailRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start gap-2">
      <span className="text-slate-400 w-24 shrink-0 text-sm">{label}</span>
      <span className={`text-white text-sm break-all ${mono ? 'font-mono' : ''}`}>{value}</span>
    </div>
  )
}
