'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { getBlockList, Block, ChainType, CHAIN_CONFIG, Pagination } from '@/lib/api'
import BlockTable from '@/components/BlockTable'
import Link from 'next/link'
import { ChevronLeft, ChevronRight, Loader2 } from 'lucide-react'

export default function BlocksPage() {
  const params = useParams()
  const chain = (params.chain as string) as ChainType
  const config = CHAIN_CONFIG[chain]

  const [blocks, setBlocks] = useState<Block[]>([])
  const [pagination, setPagination] = useState<Pagination>({ page: 1, page_size: 20, total: 0 })
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  useEffect(() => {
    let cancelled = false
    const fetchBlocks = async () => {
      setLoading(true)
      try {
        const res = await getBlockList(chain, page, 20)
        if (!cancelled) {
          setBlocks(res.blocks || [])
          setPagination(res.pagination)
        }
      } catch {
        if (!cancelled) {
          setBlocks([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchBlocks()
    return () => { cancelled = true }
  }, [chain, page])

  const totalPages = Math.ceil(pagination.total / pagination.page_size)

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div
          className="w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold"
          style={{ backgroundColor: config.color + '20', color: config.color }}
        >
          {config.icon}
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">{config.name} Blocks</h1>
          <p className="text-sm text-dark-400">Latest blocks on the {config.name} blockchain</p>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-dark-500" />
        </div>
      ) : blocks.length > 0 ? (
        <>
          <BlockTable blocks={blocks} chain={chain} />
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-4 pt-4">
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
        <div className="text-center py-24 text-dark-500">No blocks found</div>
      )}
    </div>
  )
}
