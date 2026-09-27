import React from 'react'
import ReactDOM from 'react-dom/client'
import { App as AntApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import Workspace from './workspace/Workspace'
import './styles.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#5558d9',
          colorInfo: '#5558d9',
          colorText: '#1d2740',
          colorTextSecondary: '#7b879c',
          colorBorder: '#e5e9f1',
          colorBgLayout: '#f7f8fb',
          borderRadius: 12,
          fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
        },
        components: {
          Button: { controlHeight: 38, borderRadius: 10 },
          Modal: { borderRadiusLG: 18 },
          Input: { activeBorderColor: '#888af0', hoverBorderColor: '#888af0' },
        },
      }}
    >
      <AntApp>
        <Workspace />
      </AntApp>
    </ConfigProvider>
  </React.StrictMode>,
)
