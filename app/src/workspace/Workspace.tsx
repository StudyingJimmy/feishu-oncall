import { BulbOutlined, CheckCircleFilled, CustomerServiceOutlined, LockOutlined, SearchOutlined, ThunderboltFilled } from '@ant-design/icons'
import { Avatar, Button, Tag } from 'antd'
import { useEffect, useRef, useState } from 'react'
import ConversationView from '../components/ConversationView'
import CreateOncallModal from '../components/CreateOncallModal'
import QuestionComposer from '../components/QuestionComposer'
import TicketDetail from '../components/TicketDetail'
import TicketSidebar, { MobileMenuButton } from '../components/TicketSidebar'
import type { OncallWorkspaceController } from './useOncallWorkspace'

const examples = [
  { icon: <SearchOutlined />, label: '支付接口持续超时' },
  { icon: <LockOutlined />, label: '员工登录提示权限不足' },
  { icon: <BulbOutlined />, label: '订单同步延迟' },
]

export default function Workspace({ workspace }: { workspace: OncallWorkspaceController }) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [workspace.turns, workspace.searching, workspace.activeTicketId])

  const newQuestion = () => { workspace.newQuestion(); setMobileOpen(false) }
  const selectTicket = (id: string) => { workspace.selectTicket(id); setMobileOpen(false) }

  return (
    <div className="app-shell">
      <TicketSidebar tickets={workspace.tickets} loading={workspace.ticketsLoading} activeTicketId={workspace.activeTicketId} mobileOpen={mobileOpen} onMobileOpenChange={setMobileOpen} onNew={newQuestion} onSelect={selectTicket} />
      <main className="workspace-main">
        <header className="workspace-header">
          <div className="header-left"><MobileMenuButton onClick={() => setMobileOpen(true)} /><span className="header-product">Oncall Copilot</span><span className="header-slash">/</span><span className="header-current">{workspace.activeTicket?.id ?? (workspace.home ? '发起问题' : '问题对话')}</span></div>
          <div className="header-right"><span className="header-state"><span /> 演示环境已就绪</span><Tag className="demo-tag" bordered={false}>DEMO</Tag><Avatar className="header-avatar">我</Avatar></div>
        </header>

        {workspace.home ? (
          <div className="home-scroll"><div className="home-content">
            <div className="hero-icon"><ThunderboltFilled /></div>
            <div className="hero-eyebrow">BOKE ONCALL · SMART WORKSPACE</div>
            <h1>你好，有什么需要协助？</h1>
            <p className="hero-subtitle">搜索已有方案，找到最合适的租户。复杂问题，从这里开始变简单。</p>
            <QuestionComposer value={workspace.query} onChange={workspace.setQuery} onSubmit={workspace.search} submitting={workspace.searching} matches={workspace.matches} rankingLoading={workspace.rankingLoading} onTenantSelect={workspace.selectTenant} placement="home" />
            <div className="examples"><span>试试这样提问</span><div>{examples.map((example) => <Button key={example.label} icon={example.icon} onClick={() => workspace.setQuery(example.label)}>{example.label}</Button>)}</div></div>
            <div className="hero-footer"><span><CheckCircleFilled /> 实时租户匹配</span><i /><span><CheckCircleFilled /> 知识库搜索</span><i /><span><CheckCircleFilled /> 一键发起 Oncall</span></div>
          </div></div>
        ) : (
          <>
            <div className="conversation-scroll"><div className="conversation-container">
              {workspace.activeTicket ? <TicketDetail ticket={workspace.activeTicket} /> : <ConversationView turns={workspace.turns} loading={workspace.searching} pendingQuestion={workspace.pendingQuestion} onSelectTenant={workspace.selectTenant} />}
              <div ref={endRef} />
            </div></div>
            <div className="conversation-composer-wrap"><QuestionComposer value={workspace.query} onChange={workspace.setQuery} onSubmit={workspace.search} submitting={workspace.searching} matches={workspace.matches} rankingLoading={workspace.rankingLoading} onTenantSelect={workspace.selectTenant} placement="conversation" /></div>
          </>
        )}
        <div className="floating-help"><CustomerServiceOutlined /> Oncall 工作台 · 演示版本</div>
      </main>
      <CreateOncallModal tenant={workspace.selectedTenant?.tenant ?? null} initialDescription={workspace.selectedTenant?.description ?? ''} submitting={workspace.creating} onCancel={workspace.closeTenant} onConfirm={async (description) => { await workspace.createOncall(description) }} />
    </div>
  )
}
