// ============================================================
// PriceChart 价格曲线组件（支持缩放）
// ============================================================
import { useState, useEffect } from 'react'
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, Brush } from 'recharts'
import { useChain } from '../context/ChainContext'
import { getPriceHistory } from '../api/client'
import type { PriceHistoryItem } from '../types'

export default function PriceChart() {
  const { chain } = useChain()
  const [data, setData] = useState<PriceHistoryItem[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        const resp = await getPriceHistory(chain, 200)
        setData(resp.prices || [])
      } catch {
        console.error('获取价格历史失败')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
    // 每 30 秒刷新一次
    const timer = setInterval(fetchData, 30000)
    return () => clearInterval(timer)
  }, [chain])

  if (loading) {
    return (
      <div className="bg-slate-800 rounded-lg p-6 h-64 flex items-center justify-center">
        <span className="text-slate-400">加载价格数据...</span>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="bg-slate-800 rounded-lg p-6 h-64 flex items-center justify-center">
        <span className="text-slate-400">暂无价格数据（等待价格同步...）</span>
      </div>
    )
  }

  const chartData = data
    .slice()
    .reverse() // 从旧到新
    .map((item) => ({
      time: new Date(item.timestamp * 1000).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      }),
      price: parseFloat(item.price_usd),
    }))

  const prices = chartData.map((d) => d.price)
  const minPrice = Math.min(...prices)
  const maxPrice = Math.max(...prices)
  const padding = (maxPrice - minPrice) * 0.1 || 1

  return (
    <div className="bg-slate-800 rounded-lg p-6">
      <h2 className="text-lg font-semibold text-white mb-4">
        {chain.toUpperCase()} 价格走势
        <span className="text-xs text-slate-400 ml-2">（拖动下方滑块缩放）</span>
      </h2>
      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={chartData}>
          <XAxis dataKey="time" stroke="#64748b" fontSize={12} tickLine={false} />
          <YAxis
            stroke="#64748b"
            fontSize={12}
            tickLine={false}
            domain={[minPrice - padding, maxPrice + padding]}
            tickFormatter={(v: number) => `$${v.toFixed(2)}`}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: '#1e293b',
              border: '1px solid #334155',
              borderRadius: '8px',
              color: '#e2e8f0',
            }}
            formatter={(value: number) => [`$${value.toFixed(2)}`, '价格']}
            labelStyle={{ color: '#94a3b8' }}
          />
          <Line
            type="monotone"
            dataKey="price"
            stroke="#3b82f6"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4, fill: '#3b82f6' }}
          />
          {/* 底部缩放滑块：拖动可选择显示范围 */}
          <Brush
            dataKey="time"
            height={30}
            stroke="#475569"
            fill="#1e293b"
            travellerWidth={10}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
