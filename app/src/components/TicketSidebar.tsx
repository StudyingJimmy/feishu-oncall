import { Avatar, Button, Collapse, Drawer, Skeleton, Tooltip } from 'antd'
import { CheckCircleOutlined, MenuOutlined, PlusOutlined, RightOutlined, ThunderboltFilled } from '@ant-design/icons'
import { useState } from 'react'
import type { Ticket, TicketStatus } from '../domain/types'

interface Props {
  tickets: Ticket[]
  loading: boolean
  activeTicketId: string | null
  mobileOpen: boolean
  onMobileOpenChange: (open: boolean) => void
  onNew: () => void
  onSelect: (ticketId: string) => void
}

function TicketList({ tickets, activeTicketId, onSelect }: Pick<Props, 'tickets' | 'activeTicketId' | 'onSelect'>) {
  if (!tickets.length) return <div className="sidebar-empty">暂无工单</div>
  return (
    <div className="sidebar-ticket-list">
      {tickets.map((ticket) => (
        <button
          className={`sidebar-ticket ${activeTicketId === ticket.id ? 'is-active' : ''}`}
          key={ticket.id}
          onClick={() => onSelect(ticket.id)}
          type="button"
          aria-current={activeTicketId === ticket.id ? 'page' : undefined}
        >
          <span className="sidebar-ticket-id">{ticket.id}</span>
          <span className="sidebar-ticket-title">{ticket.title}</span>
        </button>
      ))}
    </div>
  )
}

function SidebarContent({ tickets, loading, activeTicketId, onNew, onSelect }: Omit<Props, 'mobileOpen' | 'onMobileOpenChange'>) {
  const [expanded, setExpanded] = useState<string[]>(['in_progress', 'completed'])
  const statuses: { key: TicketStatus; title: string; count: number }[] = [
    { key: 'in_progress', title: '进行中', count: tickets.filter((ticket) => ticket.status === 'in_progress').length },
    { key: 'completed', title: '已完成', count: tickets.filter((ticket) => ticket.status === 'completed').length },
  ]

  return (
    <div className="sidebar-inner">
      <div className="brand-row">
        <div className="brand-icon"><ThunderboltFilled /></div>
        <div className="brand-wordmark"><strong>Boke Oncall</strong><span>智能协作工作台</span></div>
      </div>

      <Button className="new-chat-button" icon={<PlusOutlined />} onClick={onNew} block>
        发起新问题
      </Button>

      <div className="sidebar-section-label">我的 ONCALL <span>工单记录</span></div>
      {loading ? (
        <div className="sidebar-skeleton"><Skeleton active paragraph={{ rows: 5 }} title={false} /></div>
      ) : (
        <Collapse
          className="ticket-collapse"
          activeKey={expanded}
          onChange={(keys) => setExpanded(Array.isArray(keys) ? keys.map(String) : [String(keys)])}
          expandIcon={({ isActive }) => <RightOutlined rotate={isActive ? 90 : 0} />}
          ghost
          items={statuses.map((group) => ({
            key: group.key,
            label: (
              <span className="ticket-group-label">
                <span className={`group-status-dot ${group.key}`} />
                <span>{group.title}</span>
                <span className="group-count">{group.count}</span>
              </span>
            ),
            children: (
              <TicketList
                tickets={tickets.filter((ticket) => ticket.status === group.key)}
                activeTicketId={activeTicketId}
                onSelect={onSelect}
              />
            ),
          }))}
        />
      )}

      <div className="sidebar-footer">
        <div className="workspace-hint"><CheckCircleOutlined /> <span>对话、匹配与工单统一在这里处理</span></div>
        <div className="sidebar-user"><Avatar className="user-avatar">我</Avatar><span><strong>我的工作台</strong><small>演示环境</small></span><Tooltip title="当前使用模拟数据"><span className="user-online" /></Tooltip></div>
      </div>
    </div>
  )
}

export default function TicketSidebar(props: Props) {
  const contentProps = {
    tickets: props.tickets,
    loading: props.loading,
    activeTicketId: props.activeTicketId,
    onNew: props.onNew,
    onSelect: props.onSelect,
  }

  return (
    <>
      <aside className="desktop-sidebar" aria-label="工单导航"><SidebarContent {...contentProps} /></aside>
      <Drawer
        className="mobile-sidebar-drawer"
        title="Boke Oncall"
        placement="left"
        width={286}
        open={props.mobileOpen}
        onClose={() => props.onMobileOpenChange(false)}
      >
        <SidebarContent {...contentProps} />
      </Drawer>
    </>
  )
}

export function MobileMenuButton({ onClick }: { onClick: () => void }) {
  return <Button className="mobile-menu-button" type="text" icon={<MenuOutlined />} aria-label="打开工单导航" onClick={onClick} />
}
