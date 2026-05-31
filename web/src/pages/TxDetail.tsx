// ============================================================
// TxDetail 交易详情页面
// ============================================================
import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getTransactionDetail } from '../api/client'
import type { Transaction, ChainType } from '../types'

export default function TxDetail() {
  const { chain, txHash } = useParams<{ chain: string; txHash: string }>()
  const [tx, setTx] = useState<Transaction | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const fetchData = async () => {
      if (!chain || !txHash) return
      setLoading(true)
      try {
        const data = await getTransactionDetail(chain as ChainType, txHash)
        setTx(data)
      } catch {
        setError('获取交易详情失败')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [chain, txHash])

  const formatTime = (ts: number) => new Date(ts * 1000).toLocaleString('zh-CN')

  if (loading) {
    return <div className="flex justify-center items-center h-64 text-slate-400">加载中...</div>
  }
  if (error || !tx) {
    return <div className="flex justify-center items-center h-64 text-red-400">{error || '交易不存在'}</div>
  }

  return (
    <div>
      <div className="mb-6">
        <Link to="/" className="text-blue-400 hover:text-blue-300 text-sm mb-2 inline-block">← 返回首页</Link>
        <h1 className="text-2xl font-bold text-white">交易详情</h1>
      </div>
      <div className="bg-slate-800 rounded-lg p-6">
        <div className="space-y-4">
          <DetailRow label="交易哈希" value={tx.tx_hash} mono />
          <DetailRow label="状态">
            <span className={`px-2 py-1 rounded text-xs ${
              tx.status === 1 ? 'bg-green-900/50 text-green-400' : 'bg-red-900/50 text-red-400'
            }`}>
              {tx.status === 1 ? '成功' : '失败'}
            </span>
          </DetailRow>
          <DetailRow label="区块高度">
            <Link to={`/blocks/${chain}/${tx.block_number}`} className="text-blue-400 hover:text-blue-300">
              {tx.block_number}
            </Link>
          </DetailRow>
          <DetailRow label="时间戳" value={formatTime(tx.timestamp)} />
          <DetailRow label="发送方" value={tx.from_addr} mono />
          <DetailRow label="接收方" value={tx.to_addr || '(合约创建)'} mono />
          <DetailRow label="金额" value={tx.value || '0'} />
          <DetailRow label="Gas 价格" value={tx.gas_price || '-'} />
          <DetailRow label="Gas 使用" value={tx.gas_used || '-'} />
        </div>
      </div>
    </div>
  )
}

function DetailRow({ label, value, mono = false, children }: {
  label: string; value?: string; mono?: boolean; children?: React.ReactNode
}) {
  return (
    <div className="flex items-start gap-2 py-2 border-b border-slate-700/50 last:border-0">
      <span className="text-slate-400 w-24 shrink-0 text-sm">{label}</span>
      {children || <span className={`text-white text-sm break-all ${mono ? 'font-mono' : ''}`}>{value}</span>}
    </div>
  )
}
