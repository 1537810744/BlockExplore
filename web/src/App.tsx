// ============================================================
// App 组件 - 应用根组件
// ============================================================
// 负责：
//   1. 定义路由规则（URL 和页面的映射关系）
//   2. 提供全局布局（导航栏 + 内容区域）
//
// React 核心概念：
//   - 组件：可复用的 UI 单元，类似于函数
//   - JSX：JavaScript + HTML 混合语法
//   - useState：状态钩子，用于管理组件内部数据
//   - Route / Routes：定义 URL 和组件的映射
// ============================================================
import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import BlockList from './pages/BlockList'
import BlockDetail from './pages/BlockDetail'
import TxDetail from './pages/TxDetail'
import AddressTx from './pages/AddressTx'

// App 组件：应用的根组件
export default function App() {
  return (
    // Layout 包裹所有页面，提供统一的导航栏和布局
    <Layout>
      {/* Routes 定义路由规则 */}
      <Routes>
        {/* 首页：区块列表 */}
        <Route path="/" element={<BlockList />} />

        {/* 区块详情页 */}
        <Route path="/blocks/:chain/:blockNumber" element={<BlockDetail />} />

        {/* 交易详情页 */}
        <Route path="/tx/:chain/:txHash" element={<TxDetail />} />

        {/* 地址交易历史页 */}
        <Route path="/address/:chain/:address" element={<AddressTx />} />
      </Routes>
    </Layout>
  )
}
