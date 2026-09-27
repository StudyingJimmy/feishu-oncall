import { App as AntApp } from 'antd'
import { useEffect, useRef, useState } from 'react'
import type { SearchTurn, Tenant, TenantMatch, Ticket } from '../domain/types'
import { mockOncallGateway } from '../services/mockOncallGateway'

const gateway = mockOncallGateway

function makeTurnId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `turn-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function useOncallWorkspace() {
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

  const activeTicket = tickets.find((ticket) => ticket.id === activeTicketId) ?? null
  const home = !activeTicket && turns.length === 0 && !searching

  const newQuestion = () => {
    searchRequestId.current += 1
    setSearching(false)
    setPendingQuestion('')
    setTurns([])
    setActiveTicketId(null)
    setQuery('')
  }

  const selectTicket = (ticketId: string) => {
    searchRequestId.current += 1
    setSearching(false)
    setPendingQuestion('')
    setActiveTicketId(ticketId)
    setQuery('')
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

  const createOncall = async (description: string): Promise<Ticket> => {
    if (!selectedTenant) throw new Error('请先选择目标租户')
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
      return ticket
    } catch (error) {
      message.error(error instanceof Error ? error.message : '创建工单失败')
      throw error
    } finally {
      setCreating(false)
    }
  }

  return {
    tickets, ticketsLoading, query, setQuery, matches, rankingLoading,
    turns, searching, pendingQuestion, activeTicketId, activeTicket, home,
    selectedTenant, creating, newQuestion, selectTicket, selectTenant,
    search, createOncall, closeTenant: () => setSelectedTenant(null),
  }
}

export type OncallWorkspaceController = ReturnType<typeof useOncallWorkspace>
