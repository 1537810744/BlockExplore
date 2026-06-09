'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { getAddressTransactions, Transaction, ChainType, CHAIN_CONFIG, Pagination } from '@/lib/api'
import TxTable from '@/components/TxTable'
import Link from 'next/link'
import { Wallet, ChevronLeft, ChevronRight, Loader2, Copy, Check } from 'lucide-react'

export default function AddressPage() {
  const params = useParams()
  const chain = (params.chain as string) as ChainType
  const address = params.address as string
  const config = CHAIN_CONFIG[chain]

  const [txs, setTxs] = useState<Transaction[]>([])
  const [pagination, setPagination] = useState<Pagination>({ page: 1, page_size: 20, total: 0 })
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let cancelled = false
    const fetchTxs = async () => {
      setLoading(true)
      try {
        const res = await getAddressTransactions(chain, address, page, 20)
        if (!cancelled) {
          setTxs(res.transactions || [])
          setPagination(res.pagination)
        }
      } catch {
        if (!cancelled) setTxs([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchTxs()
    return () => { cancelled = true }
  }, [chain, address, page])

  const copyToClipboard = () => {
    navigator.clipboard.writeText(address)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const totalPages = Math.ceil(pagination.total / pagination.page_size)

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
          <h1 className="text-2xl font-bold text-white">Address</h1>
          <div className="flex items-center gap-2">
            <p className="text-sm text-dark-400 font-mono break-all">{address}</p>
            <button
              onClick={copyToClipboard}
              className="text-dark-500 hover:text-dark-300 transition-colors flex-shrink-0"
            >
              {copied ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
            </button>
          </div>
        </div>
      </div>

      <div className="card">
        <div className="flex items-center gap-2 mb-4">
          <Wallet className="w-5 h-5 text-primary-500" />
          <h2 className="text-lg font-semibold text-white">
            Transactions ({pagination.total})
          </h2>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-6 h-6 animate-spin text-dark-500" />
          </div>
        ) : txs.length > 0 ? (
          <>
            <TxTable transactions={txs} chain={chain} />
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-4 pt-4 mt-4 border-t border-dark-700">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="btn-ghost disabled:opacity-30 disabled:cursor-not-allowed flex items-center gap-1"
                >
                  <ChevronLeft className="w-4 h-4" /> Previous
                </button>
                <span className="text-sm text-dark-400">
                  Page {page} of {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="btn-ghost disabled:opacity-30 disabled:cursor-not-allowed flex items-center gap-1"
                >
                  Next <ChevronRight className="w-4 h-4" />
                </button>
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-12 text-dark-500">No transactions found for this address</div>
        )}
      </div>
    </div>
  )
}
