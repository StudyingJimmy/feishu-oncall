import { ArrowRightOutlined, ClockCircleOutlined, InboxOutlined } from '@ant-design/icons'
import { Empty, Segmented, Skeleton } from 'antd'
import { useState } from 'react'
import type { Ticket, TicketStatus } from '../domain/types'

interface Props {
  tickets: Ticket[]
  loading: boolean
  onOpen: (id: string) => void
}

export default function MobileTicketIndex({ tickets, loading, onOpen }: Props) {
  const [status, setStatus] = useState<TicketStatus>('in_progress')
  const ongoing = tickets.filter((ticket) => ticket.status === 'in_progress')
  const completed = tickets.filter((ticket) => ticket.status === 'completed')
  const visible = status === 'in_progress' ? ongoing : completed

  return (
    <div className="mobile-ticket-index">
      <div className="mobile-ticket-summary"><span className="mobile-summary-overline">MY ONCALL</span><h1>我的工单</h1><p>所有需要跟进的问题，都在这里。</p><div className="mobile-summary-stats"><span><strong>{ongoing.length}</strong> 进行中</span><i /><span><strong>{completed.length}</strong> 已完成</span></div></div>
      <div className="mobile-ticket-controls"><Segmented block value={status} onChange={(value) => setStatus(value as TicketStatus)} options={[{ label: `进行中 ${ongoing.length}`, value: 'in_progress' }, { label: `已完成 ${completed.length}`, value: 'completed' }]} /></div>
      <div className="mobile-ticket-list">
        {loading ? <Skeleton active paragraph={{ rows: 6 }} title={false} /> : visible.length ? visible.map((ticket) => (
          <button className="mobile-ticket-card" key={ticket.id} type="button" onClick={() => onOpen(ticket.id)}>
            <span className="mobile-ticket-card-top"><span>{ticket.id}</span><span className={ticket.status === 'completed' ? 'is-complete' : ''}><i />{ticket.status === 'completed' ? '已完成' : '进行中'}</span></span>
            <strong>{ticket.title}</strong>
            <span className="mobile-ticket-card-bottom"><span><InboxOutlined /> {ticket.tenantName}</span><span><ClockCircleOutlined /> {new Date(ticket.createdAt).toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })}</span><ArrowRightOutlined /></span>
          </button>
        )) : <div className="mobile-ticket-empty"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={status === 'in_progress' ? '暂无进行中的工单' : '暂无已完成的工单'} /></div>}
      </div>
    </div>
  )
}
