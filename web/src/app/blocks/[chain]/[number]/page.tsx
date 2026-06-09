'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { getBlockDetail, getBlockTransactions, Block, Transaction, ChainType, CHAIN_CONFIG, Pagination } from '@/lib/api'
import TxTable from '@/components/TxTable'
import Link from 'next/link'
import { Boxes, Clock, Fuel, Layers, ChevronLeft, ChevronRight, Loader2, Copy, Check } from 'lucide-react'

export default function BlockDetailPage() {
  const params = useParams()
  const chain = (params.chain as string) as ChainType
  const blockNumber = parseInt(params.number as string)
  const config = CHAIN_CONFIG[chain]

  const [block, setBlock] = useState<Block | null>(null)
  const [txs, setTxs] = useState<Transaction[]>([])
  const [pagination, setPagination] = useState<Pagination>({ page: 1, page_size: 20, total: 0 })
  const [loading, setLoading] = useState(true)
  const [txPage, setTxPage] = useState(1)
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const fetchData = async () => {
      setLoading(true)
      try {
        const [blockData, txData] = await Promise.all([
          getBlockDetail(chain, blockNumber),
          getBlockTransactions(chain, blockNumber, txPage, 20),
        ])
        if (!cancelled) {
          setBlock(blockData)
          setTxs(txData.transactions || [])
          setPagination(txData.pagination)
        }
      } catch {
        // ignore
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchData()
    return () => { cancelled = true }
  }, [chain, blockNumber, txPage])

  const copyToClipboard = (text: string, field: string) => {
    navigator.clipboard.writeText(text)
    setCopied(field)
    setTimeout(() => setCopied(null), 2000)
  }

  const totalPages = Math.ceil(pagination.total / pagination.page_size)

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-dark-500" />
      </div>
    )
  }

  if (!block) {
    return (
      <div className="text-center py-24">
        <h2 className="text-xl font-semibold text-white mb-2">Block Not Found</h2>
        <p className="text-dark-400">Block #{blockNumber} on {config.name} was not found.</p>
        <Link href={`/blocks/${chain}`} className="btn-primary inline-block mt-4">
          Back to Blocks
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link href={`/blocks/${chain}`} className="text-dark-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div
          className="w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold"
          style={{ backgroundColor: config.color + '20', color: config.color }}
        >
          {config.icon}
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Block #{block.block_number}</h1>
          <p className="text-sm text-dark-400">{config.name} Block Detail</p>
        </div>
      </div>

      <div className="card space-y-4">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <Boxes className="w-5 h-5 text-primary-500" /> Overview
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <DetailRow label="Block Number" value={block.block_number.toString()} />
          <DetailRow label="Timestamp" value={formatTimestamp(block.timestamp)} icon={<Clock className="w-4 h-4" />} />
          <DetailRow
            label="Block Hash"
            value={block.block_hash}
            copyable
            onCopy={() => copyToClipboard(block.block_hash, 'hash')}
            copied={copied === 'hash'}
          />
          <DetailRow
            label="Parent Hash"
            value={block.parent_hash}
            copyable
            onCopy={() => copyToClipboard(block.parent_hash, 'parent')}
            copied={copied === 'parent'}
          />
          <DetailRow label="Transactions" value={block.tx_count.toString()} />
          <DetailRow label="Gas Used" value={formatGas(block.gas_used)} icon={<Fuel className="w-4 h-4" />} />
          <DetailRow label="Gas Limit" value={formatGas(block.gas_limit)} />
          <DetailRow label="Size" value={formatSize(block.size_bytes)} icon={<Layers className="w-4 h-4" />} />
          {block.difficulty && block.difficulty !== '0' && (
            <DetailRow label="Difficulty" value={block.difficulty} />
          )}
        </div>
      </div>

      <div className="card">
        <h2 className="text-lg font-semibold text-white mb-4">
          Transactions ({block.tx_count})
        </h2>
        {txs.length > 0 ? (
          <>
            <TxTable transactions={txs} chain={chain} />
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-4 pt-4 mt-4 border-t border-dark-700">
                <button
                  onClick={() => setTxPage((p) => Math.max(1, p - 1))}
                  disabled={txPage <= 1}
                  className="btn-ghost disabled:opacity-30 disabled:cursor-not-allowed flex items-center gap-1"
                >
                  <ChevronLeft className="w-4 h-4" /> Previous
                </button>
                <span className="text-sm text-dark-400">
                  Page {txPage} of {totalPages}
                </span>
                <button
                  onClick={() => setTxPage((p) => Math.min(totalPages, p + 1))}
                  disabled={txPage >= totalPages}
                  className="btn-ghost disabled:opacity-30 disabled:cursor-not-allowed flex items-center gap-1"
                >
                  Next <ChevronRight className="w-4 h-4" />
                </button>
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-12 text-dark-500">No transactions in this block</div>
        )}
      </div>
    </div>
  )
}

function DetailRow({
  label,
  value,
  icon,
  copyable,
  onCopy,
  copied,
}: {
  label: string
  value: string
  icon?: React.ReactNode
  copyable?: boolean
  onCopy?: () => void
  copied?: boolean
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className="label flex items-center gap-1">
        {icon} {label}
      </span>
      <div className="flex items-center gap-2">
        <span className="value font-mono break-all text-sm">{value}</span>
        {copyable && onCopy && (
          <button
            onClick={onCopy}
            className="text-dark-500 hover:text-dark-300 transition-colors flex-shrink-0"
          >
            {copied ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
          </button>
        )}
      </div>
    </div>
  )
}

function formatTimestamp(ts: number): string {
  return new Date(ts * 1000).toLocaleString()
}

function formatGas(gas: string): string {
  const num = parseInt(gas)
  if (isNaN(num)) return gas
  if (num >= 1e6) return `${(num / 1e6).toFixed(1)}M`
  if (num >= 1e3) return `${(num / 1e3).toFixed(1)}K`
  return num.toLocaleString()
}

function formatSize(bytes: number): string {
  if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(2)} MB`
  if (bytes >= 1e3) return `${(bytes / 1e3).toFixed(2)} KB`
  return `${bytes} B`
}
