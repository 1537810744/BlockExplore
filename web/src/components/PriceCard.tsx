// ============================================================
// PriceCard 价格卡片组件
// 显示当前链的代币价格
// ============================================================
import { useState, useEffect } from 'react'
import { getCurrentPrice } from '../api/client'
import type { ChainType, PriceResponse } from '../types'

interface PriceCardProps {
  chain: ChainType
}

export default function PriceCard({ chain }: PriceCardProps) {
  const [price, setPrice] = useState<PriceResponse | null>(null)

  useEffect(() => {
    const fetchPrice = async () => {
      try {
        const data = await getCurrentPrice(chain)
        setPrice(data)
      } catch {
        // 价格获取失败时不显示
      }
    }
    fetchPrice()
    const timer = setInterval(fetchPrice, 30000)
    return () => clearInterval(timer)
  }, [chain])

  if (!price) return null

  return (
    <div className="flex items-center gap-4 text-sm">
      <span className="text-slate-400">{price.symbol}</span>
      <span className="text-white font-bold">
        ${price.price_usd.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
      </span>
    </div>
  )
}
