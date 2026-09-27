export type TicketStatus = 'in_progress' | 'completed'

export interface Tenant {
  id: string
  name: string
  domain: string
  summary: string
  keywords: string[]
  avatar: string
  color: string
}

export interface TenantMatch {
  tenant: Tenant
  score: number
  matchedTerms: string[]
}

export interface SearchHit {
  id: string
  title: string
  summary: string
  source: '知识库' | '历史工单'
  keywords: string[]
  updatedAt: string
  score: number
}

export interface Ticket {
  id: string
  title: string
  description: string
  tenantId: string
  tenantName: string
  status: TicketStatus
  createdAt: string
  groupName: string
  simulated: boolean
}

export interface SearchTurn {
  id: string
  question: string
  hits: SearchHit[]
  tenantMatches: TenantMatch[]
  createdAt: string
}

export interface CreateOncallInput {
  tenantId: string
  description: string
}

/** 页面只依赖此契约；接真实后端时替换实现，不在组件里调用飞书开放 API。 */
export interface OncallGateway {
  listTickets(): Promise<Ticket[]>
  rankTenants(query: string): Promise<TenantMatch[]>
  searchKnowledge(query: string): Promise<SearchHit[]>
  createOncall(input: CreateOncallInput): Promise<Ticket>
}
