// ============================================================
// SearchBar 搜索栏组件
// ============================================================
// 提供统一搜索输入框：
//   - 输入区块号 → 跳转区块详情
//   - 输入交易哈希 → 跳转交易详情
//   - 输入地址 → 跳转地址交易历史
//
// React 知识：
//   - 受控组件：input 的值由 React state 控制
//   - 表单事件：onChange（输入变化）、onSubmit（提交）
// ============================================================
import { useState, type FormEvent } from 'react'

interface SearchBarProps {
  onSearch: (keyword: string) => void  // 搜索回调
}

export default function SearchBar({ onSearch }: SearchBarProps) {
  // 管理输入框的值
  const [keyword, setKeyword] = useState('')

  // 表单提交处理
  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()  // 阻止表单默认的页面刷新行为
    if (keyword.trim()) {
      onSearch(keyword.trim())
      setKeyword('')  // 清空输入框
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex-1 max-w-xl">
      <div className="relative">
        {/* 搜索图标 */}
        <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
          🔍
        </span>

        {/* 搜索输入框 */}
        <input
          type="text"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          placeholder="搜索区块号、交易哈希或地址..."
          className="w-full pl-10 pr-4 py-2 bg-slate-700 border border-slate-600
                     rounded-lg text-white placeholder-slate-400
                     focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
        />
      </div>
    </form>
  )
}
