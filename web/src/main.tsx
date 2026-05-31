// ============================================================
// 应用入口文件
// ============================================================
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ChainProvider } from './context/ChainContext'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <ChainProvider>
        <App />
      </ChainProvider>
    </BrowserRouter>
  </React.StrictMode>,
)
