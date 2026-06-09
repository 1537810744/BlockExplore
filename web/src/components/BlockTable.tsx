'use client'

import Link from 'next/link'
import { Block, CHAIN_CONFIG, ChainType } from '@/lib/api'
import { Clock, Boxes, ArrowRight } from 'lucide-react'

interface Props {
  blocks: Block[]
  chain: ChainType
}

export default function BlockTable({ blocks, chain }: Props) {
  const config = CHAIN_CONFIG[chain]

  return (
    <div className="space-y-2">
      {blocks.map((block) => (
        <Link
          key={block.block_number}
          href={`/blocks/${chain}/${block.block_number}`}
          className="card-hover flex items-center gap-4 group"
        >
          <div className="w-10 h-10 rounded-lg bg-dark-700 flex items-center justify-center">
            <Boxes className="w-5 h-5 text-primary-500" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-primary-400 font-medium">{block.block_number}</span>
              <span className="text-dark-500 text-xs flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {formatTime(block.timestamp)}
              </span>
            </div>
            <div className="text-dark-400 text-xs mt-0.5">
              {block.tx_count} txns
            </div>
          </div>
          <div className="text-right">
            <div className="text-xs text-dark-400">Gas Used</div>
            <div className="text-sm text-dark-200">{formatGas(block.gas_used)}</div>
          </div>
          <ArrowRight className="w-4 h-4 text-dark-600 group-hover:text-dark-400 transition-colors" />
        </Link>
      ))}
    </div>
  )
}

function formatTime(timestamp: number): string {
  const now = Date.now() / 1000
  const diff = now - timestamp
  if (diff < 60) return `${Math.floor(diff)}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

function formatGas(gas: string): string {
  const num = parseInt(gas)
  if (isNaN(num)) return gas
  if (num >= 1e6) return `${(num / 1e6).toFixed(1)}M`
  if (num >= 1e3) return `${(num / 1e3).toFixed(1)}K`
  return num.toLocaleString()
}
