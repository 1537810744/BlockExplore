'use client'

import { useEffect, useState } from 'react'
import { getBlockList, CHAIN_CONFIG, ChainType, Block } from '@/lib/api'
import PriceCard from '@/components/PriceCard'
import PriceChart from '@/components/PriceChart'
import BlockTable from '@/components/BlockTable'
import Link from 'next/link'
import { ArrowRight, Loader2 } from 'lucide-react'

const chains: ChainType[] = ['eth', 'btc', 'sol']

export default function HomePage() {
  const [selectedChain, setSelectedChain] = useState<ChainType>('eth')
  const [blocks, setBlocks] = useState<Block[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const fetchBlocks = async () => {
      setLoading(true)
      try {
        const res = await getBlockList(selectedChain, 1, 10)
        if (!cancelled) setBlocks(res.blocks || [])
      } catch {
        if (!cancelled) setBlocks([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchBlocks()
    return () => { cancelled = true }
  }, [selectedChain])

  return (
    <div className="space-y-8">
      {/* Price Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {chains.map((chain) => (
          <PriceCard key={chain} chain={chain} />
        ))}
      </div>

      {/* Price Chart */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Price History</h2>
          <div className="flex gap-1">
            {chains.map((chain) => (
              <button
                key={chain}
                onClick={() => setSelectedChain(chain)}
                className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
                  selectedChain === chain
                    ? 'bg-primary-600 text-white'
                    : 'text-dark-400 hover:text-white hover:bg-dark-700'
                }`}
              >
                {CHAIN_CONFIG[chain].symbol}
              </button>
            ))}
          </div>
        </div>
        <PriceChart chain={selectedChain} />
      </div>

      {/* Latest Blocks */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Latest Blocks</h2>
          <Link
            href={`/blocks/${selectedChain}`}
            className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300 transition-colors"
          >
            View All <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-6 h-6 animate-spin text-dark-500" />
          </div>
        ) : blocks.length > 0 ? (
          <BlockTable blocks={blocks} chain={selectedChain} />
        ) : (
          <div className="text-center py-12 text-dark-500">No blocks found</div>
        )}
      </div>
    </div>
  )
}
