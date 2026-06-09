'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { getTransactionDetail, Transaction, ChainType, CHAIN_CONFIG } from '@/lib/api'
import Link from 'next/link'
import { FileText, Clock, ArrowRight, Fuel, Copy, Check, ChevronLeft, Loader2 } from 'lucide-react'

export default function TxDetailPage() {
  const params = useParams()
  const chain = (params.chain as string) as ChainType
  const txHash = params.hash as string
  const config = CHAIN_CONFIG[chain]

  const [tx, setTx] = useState<Transaction | null>(null)
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const fetchTx = async () => {
      try {
        const data = await getTransactionDetail(chain, txHash)
        if (!cancelled) setTx(data)
      } catch {
        // ignore
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchTx()
    return () => { cancelled = true }
  }, [chain, txHash])

  const copyToClipboard = (text: string, field: string) => {
    navigator.clipboard.writeText(text)
    setCopied(field)
    setTimeout(() => setCopied(null), 2000)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-dark-500" />
      </div>
    )
  }

  if (!tx) {
    return (
      <div className="text-center py-24">
        <h2 className="text-xl font-semibold text-white mb-2">Transaction Not Found</h2>
        <p className="text-dark-400">Transaction on {config.name} was not found.</p>
        <Link href="/" className="btn-primary inline-block mt-4">
          Back to Home
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link href="/" className="text-dark-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div
          className="w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold"
          style={{ backgroundColor: config.color + '20', color: config.color }}
        >
          {config.icon}
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Transaction Details</h1>
          <p className="text-sm text-dark-400">{config.name} Transaction</p>
        </div>
      </div>

      <div className="card space-y-4">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <FileText className="w-5 h-5 text-green-500" /> Transaction Info
        </h2>

        <div className="space-y-4">
          <DetailRow
            label="Transaction Hash"
            value={tx.tx_hash}
            copyable
            onCopy={() => copyToClipboard(tx.tx_hash, 'hash')}
            copied={copied === 'hash'}
          />

          <div className="flex items-center gap-4 p-4 bg-dark-900 rounded-lg">
            <div className="flex-1">
              <span className="label">From</span>
              <div className="flex items-center gap-2 mt-1">
                <Link
                  href={`/address/${chain}/${tx.from_addr}`}
                  className="font-mono text-sm text-primary-400 hover:text-primary-300 break-all"
                >
                  {tx.from_addr || 'N/A'}
                </Link>
                {tx.from_addr && (
                  <button
                    onClick={() => copyToClipboard(tx.from_addr, 'from')}
                    className="text-dark-500 hover:text-dark-300 flex-shrink-0"
                  >
                    {copied === 'from' ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
                  </button>
                )}
              </div>
            </div>
            <ArrowRight className="w-6 h-6 text-dark-600 flex-shrink-0" />
            <div className="flex-1">
              <span className="label">To</span>
              <div className="flex items-center gap-2 mt-1">
                {tx.to_addr ? (
                  <>
                    <Link
                      href={`/address/${chain}/${tx.to_addr}`}
                      className="font-mono text-sm text-primary-400 hover:text-primary-300 break-all"
                    >
                      {tx.to_addr}
                    </Link>
                    <button
                      onClick={() => copyToClipboard(tx.to_addr, 'to')}
                      className="text-dark-500 hover:text-dark-300 flex-shrink-0"
                    >
                      {copied === 'to' ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
                    </button>
                  </>
                ) : (
                  <span className="badge-info">Contract Creation</span>
                )}
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <DetailRow label="Block" value={`#${tx.block_number}`} />
            <DetailRow label="Timestamp" value={formatTimestamp(tx.timestamp)} icon={<Clock className="w-4 h-4" />} />
            <DetailRow label="Value" value={formatValue(tx.value, chain)} />
            <DetailRow label="Status" value={tx.status === 1 ? 'Success' : 'Failed'} />
            <DetailRow label="Gas Price" value={tx.gas_price} icon={<Fuel className="w-4 h-4" />} />
            <DetailRow label="Gas Used" value={tx.gas_used} />
            <DetailRow label="Gas Limit" value={tx.gas_limit} />
            <DetailRow label="Nonce" value={tx.nonce.toString()} />
          </div>

          {tx.input_data && tx.input_data !== '0x' && (
            <div>
              <span className="label">Input Data</span>
              <div className="mt-1 p-3 bg-dark-900 rounded-lg font-mono text-xs text-dark-300 break-all max-h-32 overflow-y-auto">
                {tx.input_data}
              </div>
            </div>
          )}
        </div>
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

function formatValue(value: string, chain: string): string {
  if (!value || value === '0') return `0 ${CHAIN_CONFIG[chain as ChainType]?.symbol || ''}`
  const units: Record<string, string> = { eth: 'ETH', btc: 'BTC', sol: 'SOL' }
  return `${parseFloat(value).toFixed(6)} ${units[chain] || ''}`
}
