// ============================================================
// Layout 布局组件
// ============================================================
import { Link, useNavigate } from 'react-router-dom'
import { useChain } from '../context/ChainContext'
import ChainSwitcher from './ChainSwitcher'
import SearchBar from './SearchBar'
import PriceCard from './PriceCard'

interface LayoutProps {
  children: React.ReactNode
}

export default function Layout({ children }: LayoutProps) {
  const { chain, setChain } = useChain()
  const navigate = useNavigate()

  const handleSearch = (keyword: string) => {
    if (/^\d+$/.test(keyword)) {
      navigate(`/blocks/${chain}/${keyword}`)
      return
    }
    if (/^0x[a-fA-F0-9]{64}$/.test(keyword)) {
      navigate(`/tx/${chain}/${keyword}`)
      return
    }
    if (/^0x[a-fA-F0-9]{40}$/.test(keyword)) {
      navigate(`/address/${chain}/${keyword}`)
      return
    }
    if (!isNaN(Number(keyword))) {
      navigate(`/blocks/${chain}/${keyword}`)
    }
  }

  return (
    <div className="min-h-screen bg-slate-900">
      <header className="bg-slate-800 border-b border-slate-700">
        <div className="max-w-7xl mx-auto px-4 py-3">
          <div className="flex items-center justify-between mb-3">
            <Link to="/" className="flex items-center gap-2">
              <span className="text-2xl">⛓</span>
              <span className="text-xl font-bold text-white">BlockExplore</span>
            </Link>
            <SearchBar onSearch={handleSearch} />
          </div>
          <div className="flex items-center justify-between">
            <ChainSwitcher current={chain} onChange={setChain} />
            <PriceCard chain={chain} />
          </div>
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-4 py-6">
        {children}
      </main>
    </div>
  )
}
