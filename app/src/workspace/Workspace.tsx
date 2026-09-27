import { BulbOutlined, CheckCircleFilled, CustomerServiceOutlined, LockOutlined, SearchOutlined, ThunderboltFilled } from '@ant-design/icons'
import { App as AntApp, Avatar, Button, Tag } from 'antd'
import { useEffect, useRef, useState } from 'react'
import type { SearchTurn, Tenant, TenantMatch, Ticket } from '../domain/types'
import { mockOncallGateway } from '../services/mockOncallGateway'
import ConversationView from '../components/ConversationView'
import CreateOncallModal from '../components/CreateOncallModal'
import QuestionComposer from '../components/QuestionComposer'
import TicketDetail from '../components/TicketDetail'
import TicketSidebar, { MobileMenuButton } from '../components/TicketSidebar'

const gateway = mockOncallGateway
const examples = [
  { icon: <SearchOutlined />, label: '支付接口持续超时' },
  { icon: <LockOutlined />, label: '员工登录提示权限不足' },
  { icon: <BulbOutlined />, label: '订单同步延迟' },
]

function makeTurnId(): string {
  return typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : `turn-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export default function Workspace() {
  const { message } = AntApp.useApp()
  const [tickets, setTickets] = useState<Ticket[]>([])
  const [ticketsLoading, setTicketsLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [matches, setMatches] = useState<TenantMatch[]>([])
  const [rankingLoading, setRankingLoading] = useState(false)
  const [turns, setTurns] = useState<SearchTurn[]>([])
  const [searching, setSearching] = useState(false)
  const [pendingQuestion, setPendingQuestion] = useState('')
  const [activeTicketId, setActiveTicketId] = useState<string | null>(null)
  const [selectedTenant, setSelectedTenant] = useState<{ tenant: Tenant; description: string } | null>(null)
  const [creating, setCreating] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const searchRequestId = useRef(0)

  useEffect(() => {
    let cancelled = false
    gateway.listTickets().then((result) => {
      if (!cancelled) setTickets(result)
    }).catch(() => {
      if (!cancelled) message.error('工单列表加载失败')
    }).finally(() => {
      if (!cancelled) setTicketsLoading(false)
    })
    return () => { cancelled = true }
  }, [message])

  useEffect(() => {
    let cancelled = false
    setRankingLoading(true)
    const timer = window.setTimeout(() => {
      gateway.rankTenants(query).then((result) => {
        if (!cancelled) setMatches(result)
      }).catch(() => {
        if (!cancelled) setMatches([])
      }).finally(() => {
        if (!cancelled) setRankingLoading(false)
      })
    }, query.trim() ? 140 : 0)
    return () => { cancelled = true; window.clearTimeout(timer) }
  }, [query])

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [turns, searching, activeTicketId])

  const activeTicket = tickets.find((ticket) => ticket.id === activeTicketId) ?? null
  const home = !activeTicket && turns.length === 0 && !searching

  const newQuestion = () => {
    searchRequestId.current += 1
    setSearching(false)
    setPendingQuestion('')
    setTurns([])
    setActiveTicketId(null)
    setQuery('')
    setMobileOpen(false)
  }

  const selectTicket = (ticketId: string) => {
    searchRequestId.current += 1
    setSearching(false)
    setPendingQuestion('')
    setActiveTicketId(ticketId)
    setQuery('')
    setMobileOpen(false)
  }

  const selectTenant = (match: TenantMatch, sourceQuestion?: string) => {
    setSelectedTenant({ tenant: match.tenant, description: sourceQuestion ?? query.trim() })
  }

  const search = async () => {
    const question = query.trim()
    if (!question || searching) return
    const requestId = ++searchRequestId.current
    setActiveTicketId(null)
    setSearching(true)
    setPendingQuestion(question)
    setQuery('')
    try {
      const [hits, tenantMatches] = await Promise.all([gateway.searchKnowledge(question), gateway.rankTenants(question)])
      if (requestId === searchRequestId.current) setTurns((current) => [...current, { id: makeTurnId(), question, hits, tenantMatches, createdAt: new Date().toISOString() }])
    } catch {
      if (requestId === searchRequestId.current) {
        message.error('搜索失败，请稍后重试')
        setQuery(question)
      }
    } finally {
      if (requestId === searchRequestId.current) {
        setSearching(false)
        setPendingQuestion('')
      }
    }
  }

  const createOncall = async (description: string) => {
    if (!selectedTenant) return
    setCreating(true)
    try {
      const ticket = await gateway.createOncall({ tenantId: selectedTenant.tenant.id, description })
      searchRequestId.current += 1
      setSearching(false)
      setPendingQuestion('')
      setTickets((current) => [ticket, ...current])
      setActiveTicketId(ticket.id)
      setSelectedTenant(null)
      setQuery('')
      message.success(`已模拟拉群并创建工单 ${ticket.id}`)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '创建工单失败')
      throw error
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="app-shell">
      <TicketSidebar tickets={tickets} loading={ticketsLoading} activeTicketId={activeTicketId} mobileOpen={mobileOpen} onMobileOpenChange={setMobileOpen} onNew={newQuestion} onSelect={selectTicket} />
      <main className="workspace-main">
        <header className="workspace-header">
          <div className="header-left"><MobileMenuButton onClick={() => setMobileOpen(true)} /><span className="header-product">Oncall Copilot</span><span className="header-slash">/</span><span className="header-current">{activeTicket?.id ?? (home ? '发起问题' : '问题对话')}</span></div>
          <div className="header-right"><span className="header-state"><span /> 演示环境已就绪</span><Tag className="demo-tag" bordered={false}>DEMO</Tag><Avatar className="header-avatar">我</Avatar></div>
        </header>

        {home ? (
          <div className="home-scroll"><div className="home-content">
            <div className="hero-icon"><ThunderboltFilled /></div>
            <div className="hero-eyebrow">BOKE ONCALL · SMART WORKSPACE</div>
            <h1>你好，有什么需要协助？</h1>
            <p className="hero-subtitle">搜索已有方案，找到最合适的租户。复杂问题，从这里开始变简单。</p>
            <QuestionComposer value={query} onChange={setQuery} onSubmit={search} submitting={searching} matches={matches} rankingLoading={rankingLoading} onTenantSelect={selectTenant} placement="home" />
            <div className="examples"><span>试试这样提问</span><div>{examples.map((example) => <Button key={example.label} icon={example.icon} onClick={() => setQuery(example.label)}>{example.label}</Button>)}</div></div>
            <div className="hero-footer"><span><CheckCircleFilled /> 实时租户匹配</span><i /><span><CheckCircleFilled /> 知识库搜索</span><i /><span><CheckCircleFilled /> 一键发起 Oncall</span></div>
          </div></div>
        ) : (
          <>
            <div className="conversation-scroll"><div className="conversation-container">
              {activeTicket ? <TicketDetail ticket={activeTicket} /> : <ConversationView turns={turns} loading={searching} pendingQuestion={pendingQuestion} onSelectTenant={selectTenant} />}
              <div ref={endRef} />
            </div></div>
            <div className="conversation-composer-wrap"><QuestionComposer value={query} onChange={setQuery} onSubmit={search} submitting={searching} matches={matches} rankingLoading={rankingLoading} onTenantSelect={selectTenant} placement="conversation" /></div>
          </>
        )}
        <div className="floating-help"><CustomerServiceOutlined /> Oncall 工作台 · 演示版本</div>
      </main>
      <CreateOncallModal tenant={selectedTenant?.tenant ?? null} initialDescription={selectedTenant?.description ?? ''} submitting={creating} onCancel={() => setSelectedTenant(null)} onConfirm={createOncall} />
    </div>
  )
}
