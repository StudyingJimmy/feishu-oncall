import type { CreateOncallInput, OncallGateway, SearchHit, Tenant, TenantMatch, Ticket } from '../domain/types'

const STORAGE_KEY = 'boke-oncall:demo-tickets:v2'

export const tenants: Tenant[] = [
  { id: 'xinghe', name: '星河零售', domain: '交易与支付', summary: '支付网关、交易链路与退款结算', keywords: ['支付', '交易', '结算', '退款', '收银', '接口', '超时'], avatar: '星', color: '#e8e9ff' },
  { id: 'yuanshan', name: '远山科技', domain: '账号与权限', summary: '员工账号、单点登录与组织权限', keywords: ['登录', '账号', '权限', '认证', '员工', '密码', '单点'], avatar: '远', color: '#e4f1ff' },
  { id: 'qinglan', name: '青岚物流', domain: '履约与配送', summary: '订单履约、配送状态与仓储', keywords: ['订单', '物流', '配送', '运单', '发货', '履约', '仓库', '同步', '延迟'], avatar: '青', color: '#e2f4ef' },
  { id: 'mingzhou', name: '明舟数据', domain: '数据与报表', summary: '数据同步、指标看板与报表查询', keywords: ['数据', '报表', '同步', '统计', '看板', '查询', '延迟'], avatar: '明', color: '#fff0e5' },
  { id: 'beichen', name: '北辰云服', domain: '平台与基础设施', summary: '云资源、网络连接与服务部署', keywords: ['服务器', '网络', '连接', '部署', '故障', '告警', '服务'], avatar: '北', color: '#f0eaff' },
]

const articles: Omit<SearchHit, 'score'>[] = [
  { id: 'kb-1042', title: '支付接口超时的自查步骤', source: '知识库', summary: '从请求 ID、网关状态和上游响应时间三个方向定位超时，并记录调用链。', keywords: ['支付', '接口', '超时', '交易'], updatedAt: '最近更新' },
  { id: 'kb-1088', title: '登录失败与权限不足排查', source: '知识库', summary: '核对账号状态、SSO 会话和角色权限，附上报错时间及用户 ID。', keywords: ['登录', '账号', '权限', '认证'], updatedAt: '本周更新' },
  { id: 'oc-2019', title: '订单状态同步延迟处理记录', source: '历史工单', summary: '检查同步任务和消息队列积压，确认受影响订单及恢复时间。', keywords: ['订单', '同步', '延迟', '履约'], updatedAt: '历史案例' },
  { id: 'kb-1097', title: '服务告警与网络异常检查', source: '知识库', summary: '确认告警级别和影响范围，排查链路连通性及最近部署。', keywords: ['服务', '网络', '故障', '告警', '部署'], updatedAt: '本月更新' },
  { id: 'oc-2033', title: '经营报表数据不一致', source: '历史工单', summary: '核对数据刷新时间、筛选条件与上游计算任务运行记录。', keywords: ['数据', '报表', '看板', '统计'], updatedAt: '历史案例' },
]

const pause = (milliseconds: number) => new Promise<void>((resolve) => window.setTimeout(resolve, milliseconds))

function offsetDate(days: number, hour: number, minute: number): string {
  const date = new Date()
  date.setDate(date.getDate() - days)
  date.setHours(hour, minute, 0, 0)
  return date.toISOString()
}

const sampleTickets: Ticket[] = [
  { id: 'OC-20260927-018', title: '支付接口响应超时', description: '支付接口偶发超时，影响部分交易请求。', tenantId: 'xinghe', tenantName: '星河零售', status: 'in_progress', createdAt: offsetDate(0, 9, 42), groupName: '星河零售 · Oncall 协作群', simulated: true },
  { id: 'OC-20260926-042', title: '订单同步延迟', description: '订单状态同步比预期晚约 20 分钟。', tenantId: 'qinglan', tenantName: '青岚物流', status: 'in_progress', createdAt: offsetDate(1, 16, 18), groupName: '青岚物流 · Oncall 协作群', simulated: true },
  { id: 'OC-20260925-031', title: '员工登录失败', description: '部分员工登录后提示权限不足。', tenantId: 'yuanshan', tenantName: '远山科技', status: 'completed', createdAt: offsetDate(2, 11, 30), groupName: '远山科技 · Oncall 协作群', simulated: true },
  { id: 'OC-20260923-012', title: '报表数据不一致', description: '经营看板与明细数据不一致。', tenantId: 'mingzhou', tenantName: '明舟数据', status: 'completed', createdAt: offsetDate(4, 14, 5), groupName: '明舟数据 · Oncall 协作群', simulated: true },
]

function storedTickets(): Ticket[] {
  try {
    const value: unknown = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '[]')
    return Array.isArray(value) ? value.filter((ticket): ticket is Ticket => typeof ticket?.id === 'string' && typeof ticket?.tenantId === 'string') : []
  } catch {
    return []
  }
}

function persist(tickets: Ticket[]): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(tickets))
  } catch {
    // 隐私模式或存储被禁用时，本次交互仍然可完成。
  }
}

function relevance(query: string, text: string, keywords: string[]): number {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return 0
  let score = text.toLowerCase().includes(normalized) ? 24 : 0
  for (const keyword of keywords) {
    if (normalized.includes(keyword.toLowerCase())) score += 11 + keyword.length
    else if (keyword.includes(normalized)) score += 4
  }
  for (const character of new Set(normalized.replace(/\s/g, ''))) {
    if (text.toLowerCase().includes(character)) score += 0.3
  }
  return score
}

export function rankTenantMatches(query: string): TenantMatch[] {
  return tenants
    .map((tenant) => ({
      tenant,
      score: relevance(query, `${tenant.name} ${tenant.domain} ${tenant.summary}`, tenant.keywords),
      matchedTerms: tenant.keywords.filter((word) => query.toLowerCase().includes(word.toLowerCase())),
    }))
    .sort((left, right) => right.score - left.score || tenants.indexOf(left.tenant) - tenants.indexOf(right.tenant))
    .slice(0, 4)
}

export const mockOncallGateway: OncallGateway = {
  async listTickets() {
    await pause(140)
    return [...storedTickets(), ...sampleTickets]
  },
  async rankTenants(query) {
    await pause(90)
    return rankTenantMatches(query)
  },
  async searchKnowledge(query) {
    await pause(480)
    return articles
      .map((article) => ({ ...article, score: relevance(query, `${article.title} ${article.summary}`, article.keywords) }))
      .sort((left, right) => right.score - left.score)
      .filter((article) => article.score >= 4)
      .slice(0, 3)
  },
  async createOncall(input: CreateOncallInput) {
    await pause(720)
    const tenant = tenants.find((item) => item.id === input.tenantId)
    if (!tenant) throw new Error('目标租户不存在')
    const description = input.description.trim()
    if (!description) throw new Error('请填写问题描述')

    const now = new Date()
    const day = `${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}`
    const sequence = [...sampleTickets, ...storedTickets()]
      .filter((ticket) => ticket.id.startsWith(`OC-${day}-`))
      .reduce((max, ticket) => Math.max(max, Number(ticket.id.slice(-3)) || 0), 0) + 1
    const ticket: Ticket = {
      id: `OC-${day}-${String(sequence).padStart(3, '0')}`,
      title: description.slice(0, 20),
      description,
      tenantId: tenant.id,
      tenantName: tenant.name,
      status: 'in_progress',
      createdAt: now.toISOString(),
      groupName: `${tenant.name} · Oncall 协作群（模拟）`,
      simulated: true,
    }
    persist([ticket, ...storedTickets()])
    return ticket
  },
}
