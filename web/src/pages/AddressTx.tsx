// ============================================================
// AddressTx 地址交易历史页面
// ============================================================
import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getAddressTransactions } from '../api/client'
import type { Transaction, ChainType } from '../types'

export default function AddressTx() {
  const { chain, address } = useParams<{ chain: string; address: string }>()
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const pageSize = 20

  useEffect(() => {
    const fetchData = async () => {
      if (!chain || !address) return
      setLoading(true)
      try {
        const data = await getAddressTransactions(chain as ChainType, address, page, pageSize)
        setTransactions(data.transactions || [])
        setTotal(data.pagination?.total || 0)
      } catch {
        console.error('获取地址交易失败')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [chain, address, page])

  const shorten = (s: string, n = 10) => s ? `${s.slice(0, n)}...${s.slice(-8)}` : '-'
  const formatTime = (ts: number) => new Date(ts * 1000).toLocaleString('zh-CN')
  const totalPages = Math.ceil(total / pageSize)

  if (loading) {
    return <div className="flex justify-center items-center h-64 text-slate-400">加载中...</div>
  }

  return (
    <div>
      <div className="mb-6">
        <Link to="/" className="text-blue-400 hover:text-blue-300 text-sm mb-2 inline-block">← 返回首页</Link>
        <h1 className="text-2xl font-bold text-white">地址交易历史</h1>
        <p className="text-slate-400 font-mono text-sm mt-1">{address}</p>
      </div>
      <div className="bg-slate-800 rounded-lg overflow-hidden">
        {transactions.length === 0 ? (
          <div className="px-6 py-8 text-center text-slate-400">暂无交易记录</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700 text-slate-400 text-sm">
                <th className="px-6 py-3 text-left">交易哈希</th>
                <th className="px-6 py-3 text-left">区块</th>
                <th className="px-6 py-3 text-left">从</th>
                <th className="px-6 py-3 text-left">到</th>
                <th className="px-6 py-3 text-left">金额</th>
                <th className="px-6 py-3 text-left">时间</th>
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
                  <td className="px-6 py-4">
                    <Link to={`/blocks/${chain}/${tx.block_number}`} className="text-blue-400 hover:text-blue-300 text-sm">
                      {tx.block_number}
                    </Link>
                  </td>
                  <td className="px-6 py-4 font-mono text-sm text-slate-300">{shorten(tx.from_addr)}</td>
                  <td className="px-6 py-4 font-mono text-sm text-slate-300">
                    {tx.to_addr ? shorten(tx.to_addr) : '-'}
                  </td>
                  <td className="px-6 py-4 text-sm text-slate-300">{tx.value || '0'}</td>
                  <td className="px-6 py-4 text-sm text-slate-300">{formatTime(tx.timestamp)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      {totalPages > 1 && (
        <div className="flex justify-center items-center gap-4 mt-6">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}
            className="px-4 py-2 bg-slate-700 rounded-lg text-sm hover:bg-slate-600 disabled:opacity-50">
            上一页
          </button>
          <span className="text-slate-400">第 {page} / {totalPages} 页</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}
            className="px-4 py-2 bg-slate-700 rounded-lg text-sm hover:bg-slate-600 disabled:opacity-50">
            下一页
          </button>
        </div>
      )}
    </div>
  )
}
