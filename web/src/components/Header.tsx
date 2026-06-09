'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { CHAIN_CONFIG, ChainType } from '@/lib/api'
import SearchBar from './SearchBar'

const chains: ChainType[] = ['eth', 'btc', 'sol']

export default function Header() {
  const pathname = usePathname()

  return (
    <header className="sticky top-0 z-50 bg-dark-900/80 backdrop-blur-xl border-b border-dark-800">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center gap-8">
            <Link href="/" className="flex items-center gap-2 text-xl font-bold text-white">
              <span className="text-primary-500">Block</span>
              <span>Explore</span>
            </Link>
            <nav className="hidden md:flex items-center gap-1">
              {chains.map((chain) => {
                const config = CHAIN_CONFIG[chain]
                const isActive = pathname.includes(`/${chain}`)
                return (
                  <Link
                    key={chain}
                    href={`/blocks/${chain}`}
                    className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-dark-700 text-white'
                        : 'text-dark-400 hover:text-white hover:bg-dark-800'
                    }`}
                  >
                    <span className="mr-1.5">{config.icon}</span>
                    {config.name}
                  </Link>
                )
              })}
            </nav>
          </div>
          <div className="w-96">
            <SearchBar />
          </div>
        </div>
      </div>
    </header>
  )
}
