// ============================================================
// BlockList 区块列表页面（首页）
// ============================================================
import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useChain } from '../context/ChainContext'
import { getBlockList } from '../api/client'
import PriceChart from '../components/PriceChart'
import type { Block } from '../types'

export default function BlockList() {
  const { chain } = useChain()
  const [blocks, setBlocks] = useState<Block[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const pageSize = 20

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        const data = await getBlockList(chain, page, pageSize)
        setBlocks(data.blocks || [])
        setTotal(data.pagination?.total || 0)
      } catch (err) {
        console.error('获取区块列表失败:', err)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [chain, page])

  // 切换链时重置页码
  useEffect(() => {
    setPage(1)
  }, [chain])

  const totalPages = Math.ceil(total / pageSize)
  const formatTime = (ts: number) => new Date(ts * 1000).toLocaleString('zh-CN')

  return (
    <div>
      {/* 价格曲线 */}
      <div className="mb-6">
        <PriceChart />
      </div>

      {/* 区块列表 */}
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold text-white">最新区块</h1>
        <span className="text-slate-400">共 {total} 个区块</span>
      </div>

      {loading ? (
        <div className="flex justify-center items-center h-32">
          <div className="text-slate-400">加载中...</div>
        </div>
      ) : (
        <>
          <div className="bg-slate-800 rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700 text-slate-400 text-sm">
                  <th className="px-6 py-3 text-left">区块高度</th>
                  <th className="px-6 py-3 text-left">时间</th>
                  <th className="px-6 py-3 text-left">交易数</th>
                  <th className="px-6 py-3 text-left">矿工</th>
                  <th className="px-6 py-3 text-left">Gas 使用</th>
                </tr>
              </thead>
              <tbody>
                {blocks.map((block) => (
                  <tr key={block.block_number} className="border-b border-slate-700/50 hover:bg-slate-700/30 transition-colors">
                    <td className="px-6 py-4">
                      <Link to={`/blocks/${chain}/${block.block_number}`} className="text-blue-400 hover:text-blue-300 font-mono">
                        {block.block_number}
                      </Link>
                    </td>
                    <td className="px-6 py-4 text-slate-300 text-sm">{formatTime(block.timestamp)}</td>
                    <td className="px-6 py-4">
                      <span className="bg-slate-700 px-2 py-1 rounded text-sm">{block.tx_count} 笔</span>
                    </td>
                    <td className="px-6 py-4 font-mono text-sm text-slate-300">
                      {block.miner ? `${block.miner.slice(0, 8)}...${block.miner.slice(-6)}` : '-'}
                    </td>
                    <td className="px-6 py-4 text-sm text-slate-300">{block.gas_used || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex justify-center items-center gap-4 mt-6">
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
  )
}
