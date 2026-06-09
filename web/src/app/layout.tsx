import type { Metadata } from 'next'
import './globals.css'
import Header from '@/components/Header'

export const metadata: Metadata = {
  title: 'BlockExplore - Multi-Chain Blockchain Explorer',
  description: 'Explore Ethereum, Bitcoin, and Solana blockchain data',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-dark-950 text-dark-50">
        <Header />
        <main className="max-w-7xl mx-auto px-4 py-6">
          {children}
        </main>
        <footer className="border-t border-dark-800 py-6 mt-12">
          <div className="max-w-7xl mx-auto px-4 text-center text-dark-500 text-sm">
            BlockExplore - Multi-Chain Blockchain Explorer
          </div>
        </footer>
      </body>
    </html>
  )
}
