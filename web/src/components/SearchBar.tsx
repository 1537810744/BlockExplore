'use client'

import { useState, FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { Search } from 'lucide-react'

export default function SearchBar() {
  const [query, setQuery] = useState('')
  const router = useRouter()

  const handleSearch = (e: FormEvent) => {
    e.preventDefault()
    const q = query.trim()
    if (!q) return

    if (/^0x[a-fA-F0-9]{64}$/.test(q)) {
      router.push(`/tx/eth/${q}`)
    } else if (/^(0x[a-fA-F0-9]{40}|[a-fA-F0-9]{40})$/.test(q)) {
      router.push(`/address/eth/${q}`)
    } else if (/^\d+$/.test(q)) {
      router.push(`/blocks/eth/${q}`)
    } else if (q.length === 64 || /^[a-fA-F0-9]{64}$/.test(q)) {
      router.push(`/tx/btc/${q}`)
    } else if (/^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$/.test(q) || /^bc1[a-z0-9]{39,59}$/.test(q)) {
      router.push(`/address/btc/${q}`)
    } else if (/^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(q)) {
      router.push(`/address/sol/${q}`)
    } else {
      router.push(`/blocks/eth?q=${encodeURIComponent(q)}`)
    }
  }

  return (
    <form onSubmit={handleSearch} className="relative">
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-500" />
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search by address / tx hash / block number"
        className="w-full bg-dark-800 border border-dark-700 rounded-lg pl-10 pr-4 py-2 text-sm text-dark-100 placeholder:text-dark-500 focus:outline-none focus:border-primary-600 focus:ring-1 focus:ring-primary-600 transition-colors"
      />
    </form>
  )
}
