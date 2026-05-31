// ============================================================
// ChainContext 链上下文
// 在整个应用中共享当前选中的链
// ============================================================
import { createContext, useContext, useState, type ReactNode } from 'react'
import type { ChainType } from '../types'

// 上下文类型
interface ChainContextType {
  chain: ChainType
  setChain: (chain: ChainType) => void
}

// 创建上下文（默认值）
const ChainContext = createContext<ChainContextType>({
  chain: 'eth',
  setChain: () => {},
})

// 自定义 Hook：方便使用上下文
export function useChain() {
  return useContext(ChainContext)
}

// Provider 组件
export function ChainProvider({ children }: { children: ReactNode }) {
  const [chain, setChain] = useState<ChainType>('eth')

  return (
    <ChainContext.Provider value={{ chain, setChain }}>
      {children}
    </ChainContext.Provider>
  )
}
