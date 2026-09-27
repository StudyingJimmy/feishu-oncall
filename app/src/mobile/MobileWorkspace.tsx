import { ArrowLeftOutlined, BookOutlined, CheckCircleOutlined, FileSearchOutlined, MessageOutlined, PlusOutlined, ThunderboltFilled, UnorderedListOutlined } from '@ant-design/icons'
import { Avatar, Button, Tag } from 'antd'
import { useEffect, useRef, useState } from 'react'
import ConversationView from '../components/ConversationView'
import TicketDetail from '../components/TicketDetail'
import type { OncallWorkspaceController } from '../workspace/useOncallWorkspace'
import MobileComposer from './MobileComposer'
import MobileCreateOncallDrawer from './MobileCreateOncallDrawer'
import MobileTicketIndex from './MobileTicketIndex'

type MobileTab = 'chat' | 'tickets'

const prompts = [
  { icon: <FileSearchOutlined />, text: '支付接口持续超时', tone: 'violet' },
  { icon: <BookOutlined />, text: '员工登录提示权限不足', tone: 'blue' },
  { icon: <CheckCircleOutlined />, text: '订单同步延迟', tone: 'green' },
]

export default function MobileWorkspace({ workspace }: { workspace: OncallWorkspaceController }) {
  const [tab, setTab] = useState<MobileTab>('chat')
  const [detailId, setDetailId] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const ticket = workspace.tickets.find((item) => item.id === detailId) ?? null
  const chatEmpty = workspace.turns.length === 0 && !workspace.searching

  useEffect(() => {
    if (tab === 'chat' && !chatEmpty) {
      const element = scrollRef.current
      element?.scrollTo({ top: element.scrollHeight, behavior: 'smooth' })
    }
  }, [tab, chatEmpty, workspace.turns, workspace.searching])

  const newQuestion = () => {
    workspace.newQuestion()
    setTab('chat')
    setDetailId(null)
  }

  const openTicket = (id: string) => {
    workspace.selectTicket(id)
    setDetailId(id)
    setTab('tickets')
  }

  const createOncall = async (description: string) => {
    const created = await workspace.createOncall(description)
    setDetailId(created.id)
    setTab('tickets')
  }

  const search = () => {
    setTab('chat')
    setDetailId(null)
    void workspace.search()
  }

  return (
    <div className="mobile-preview-stage">
      <div className="mobile-app">
        <header className="mobile-app-header">
          {ticket && tab === 'tickets' ? (
            <><Button type="text" icon={<ArrowLeftOutlined />} aria-label="返回工单列表" onClick={() => setDetailId(null)} /><div className="mobile-header-title"><strong>工单详情</strong><span>{ticket.id}</span></div></>
          ) : (
            <><div className="mobile-brand-icon"><ThunderboltFilled /></div><div className="mobile-header-title"><strong>{tab === 'tickets' ? '我的 Oncall' : 'Boke Oncall'}</strong><span>{tab === 'tickets' ? '工单记录' : '智能协作工作台'}</span></div></>
          )}
          <Tag className="mobile-demo-tag" bordered={false}>DEMO</Tag>
          <Avatar className="mobile-header-avatar">我</Avatar>
        </header>

        <main ref={scrollRef} className={`mobile-screen ${tab === 'chat' ? 'chat-screen' : 'tickets-screen'}`}>
          {tab === 'chat' ? (
            chatEmpty ? (
              <div className="mobile-home">
                <div className="mobile-home-orb"><ThunderboltFilled /></div>
                <span className="mobile-home-eyebrow">ONCALL COPILOT</span>
                <h1>今天有什么<br />需要协助？</h1>
                <p>描述遇到的问题，查找已有方案，<br />快速联系最合适的服务租户。</p>
                <div className="mobile-prompt-section"><span>快速开始</span>{prompts.map((prompt) => <button type="button" key={prompt.text} className="mobile-prompt" onClick={() => workspace.setQuery(prompt.text)}><span className={`mobile-prompt-icon ${prompt.tone}`}>{prompt.icon}</span><strong>{prompt.text}</strong><span className="mobile-prompt-arrow">↗</span></button>)}</div>
                <button className="mobile-flow-card" type="button" onClick={() => setTab('tickets')}><span>我的 Oncall</span><strong>{workspace.tickets.filter((item) => item.status === 'in_progress').length} 张工单正在处理中</strong><small>点击查看工单进展 →</small></button>
              </div>
            ) : (
              <div className="mobile-conversation"><ConversationView turns={workspace.turns} loading={workspace.searching} pendingQuestion={workspace.pendingQuestion} onSelectTenant={workspace.selectTenant} /></div>
            )
          ) : ticket ? (
            <div className="mobile-ticket-detail"><TicketDetail ticket={ticket} /></div>
          ) : (
            <MobileTicketIndex tickets={workspace.tickets} loading={workspace.ticketsLoading} onOpen={openTicket} />
          )}
        </main>

        {tab === 'chat' && <MobileComposer value={workspace.query} onChange={workspace.setQuery} onSubmit={search} submitting={workspace.searching} matches={workspace.matches} rankingLoading={workspace.rankingLoading} onSelectTenant={workspace.selectTenant} />}

        <nav className="mobile-bottom-nav" aria-label="移动端主导航">
          <button type="button" className={tab === 'chat' ? 'is-active' : ''} aria-current={tab === 'chat' ? 'page' : undefined} onClick={() => { setTab('chat'); setDetailId(null) }}><MessageOutlined /><span>对话</span></button>
          <button type="button" className="mobile-nav-new" aria-label="发起新问题" onClick={newQuestion}><span><PlusOutlined /></span><small>新问题</small></button>
          <button type="button" className={tab === 'tickets' ? 'is-active' : ''} aria-current={tab === 'tickets' ? 'page' : undefined} onClick={() => { setTab('tickets'); setDetailId(null) }}><UnorderedListOutlined /><span>工单</span></button>
        </nav>

        <MobileCreateOncallDrawer tenant={workspace.selectedTenant?.tenant ?? null} initialDescription={workspace.selectedTenant?.description ?? ''} submitting={workspace.creating} onCancel={workspace.closeTenant} onConfirm={createOncall} />
      </div>
    </div>
  )
}
