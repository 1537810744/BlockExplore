'use client'

import Link from 'next/link'
import { Transaction, CHAIN_CONFIG, ChainType } from '@/lib/api'
import { ArrowRight, FileText } from 'lucide-react'

interface Props {
  transactions: Transaction[]
  chain: ChainType
}

export default function TxTable({ transactions, chain }: Props) {
  const config = CHAIN_CONFIG[chain]

  return (
    <div className="space-y-2">
      {transactions.map((tx) => (
        <Link
          key={tx.tx_hash}
          href={`/tx/${chain}/${tx.tx_hash}`}
          className="card-hover flex items-center gap-4 group"
        >
          <div className="w-10 h-10 rounded-lg bg-dark-700 flex items-center justify-center">
            <FileText className="w-5 h-5 text-green-500" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm text-primary-400 font-mono truncate">
              {tx.tx_hash.slice(0, 18)}...{tx.tx_hash.slice(-6)}
            </div>
            <div className="flex items-center gap-2 mt-0.5 text-xs text-dark-400">
              <span className="font-mono">From {tx.from_addr ? tx.from_addr.slice(0, 10) + '...' : 'N/A'}</span>
              <ArrowRight className="w-3 h-3" />
              <span className="font-mono">To {tx.to_addr ? tx.to_addr.slice(0, 10) + '...' : 'N/A'}</span>
            </div>
          </div>
          <div className="text-right">
            <div className="text-sm font-medium text-dark-100">
              {formatValue(tx.value, chain)}
            </div>
            <div className="text-xs text-dark-500">Block #{tx.block_number}</div>
          </div>
          <ArrowRight className="w-4 h-4 text-dark-600 group-hover:text-dark-400 transition-colors" />
        </Link>
      ))}
    </div>
  )
}

function formatValue(value: string, chain: string): string {
  if (!value || value === '0') return `0 ${CHAIN_CONFIG[chain as ChainType]?.symbol || ''}`
  const units: Record<string, string> = { eth: 'ETH', btc: 'BTC', sol: 'SOL' }
  return `${parseFloat(value).toFixed(6)} ${units[chain] || ''}`
}
