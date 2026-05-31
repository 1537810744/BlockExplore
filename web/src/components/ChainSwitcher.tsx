// ============================================================
// ChainSwitcher 链切换组件
// ============================================================
// 显示 BTC / ETH / SOL 三个按钮，点击切换当前链
// 选中的按钮高亮显示
//
// React 知识：
//   - props：父组件传递给子组件的数据
//   - 条件渲染：根据条件显示不同的样式
// ============================================================
import type { ChainType } from '../types'

// 组件的 props 类型定义
interface ChainSwitcherProps {
  current: ChainType                // 当前选中的链
  onChange: (chain: ChainType) => void  // 切换链时的回调函数
}

// 链配置：每条链的显示名称和颜色
const chains: { key: ChainType; label: string; color: string }[] = [
  { key: 'eth', label: 'Ethereum', color: 'bg-blue-600' },
  { key: 'btc', label: 'Bitcoin', color: 'bg-orange-600' },
  { key: 'sol', label: 'Solana', color: 'bg-purple-600' },
]

export default function ChainSwitcher({ current, onChange }: ChainSwitcherProps) {
  return (
    <div className="flex gap-2">
      {/* 遍历链配置，为每条链创建一个按钮 */}
      {chains.map((c) => (
        <button
          key={c.key}
          // onClick 调用父组件传入的 onChange 函数
          onClick={() => onChange(c.key)}
          className={`
            px-4 py-2 rounded-lg font-medium text-sm transition-all
            ${current === c.key
              ? `${c.color} text-white shadow-lg`   // 选中状态：彩色背景
              : 'bg-slate-700 text-slate-300 hover:bg-slate-600'  // 未选中：灰色背景
            }
          `}
        >
          {c.label}
        </button>
      ))}
    </div>
  )
}
