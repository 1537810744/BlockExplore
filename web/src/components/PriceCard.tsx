'use client'

import { useEffect, useState } from 'react'
import { getCurrentPrice, CHAIN_CONFIG, ChainType, PriceResponse } from '@/lib/api'
import { TrendingUp, TrendingDown, Loader2 } from 'lucide-react'

interface Props {
  chain: ChainType
}

export default function PriceCard({ chain }: Props) {
  const [price, setPrice] = useState<PriceResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const config = CHAIN_CONFIG[chain]

  useEffect(() => {
    let cancelled = false
    const fetchPrice = async () => {
      try {
        const data = await getCurrentPrice(chain)
        if (!cancelled) setPrice(data)
      } catch {
        // ignore
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchPrice()
    const interval = setInterval(fetchPrice, 120000)
    return () => { cancelled = true; clearInterval(interval) }
  }, [chain])

  return (
    <div className="card-hover">
      <div className="flex items-center gap-3 mb-3">
        <div
          className="w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold"
          style={{ backgroundColor: config.color + '20', color: config.color }}
        >
          {config.icon}
        </div>
        <div>
          <div className="font-medium text-white">{config.name}</div>
          <div className="text-xs text-dark-400">{config.symbol}</div>
        </div>
      </div>
      {loading ? (
        <div className="flex items-center justify-center py-4">
          <Loader2 className="w-5 h-5 animate-spin text-dark-500" />
        </div>
      ) : price ? (
        <div>
          <div className="text-2xl font-bold text-white">
            ${price.price_usd.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
          <div className="flex items-center gap-1 mt-1 text-xs text-dark-400">
            <TrendingUp className="w-3 h-3 text-green-500" />
            <span>USD</span>
          </div>
        </div>
      ) : (
        <div className="text-sm text-dark-500">Price unavailable</div>
      )}
    </div>
  )
}
