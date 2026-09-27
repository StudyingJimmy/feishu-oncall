import { CheckCircleFilled, ClockCircleOutlined, FileTextOutlined, TeamOutlined, ThunderboltFilled } from '@ant-design/icons'
import { Avatar, Card, Descriptions, Tag, Timeline } from 'antd'
import type { Ticket } from '../domain/types'

export default function TicketDetail({ ticket }: { ticket: Ticket }) {
  const completed = ticket.status === 'completed'
  const date = new Date(ticket.createdAt).toLocaleString('zh-CN', { month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit' })

  return (
    <div className="ticket-conversation">
      <div className="user-message"><span>{ticket.description}</span><Avatar className="chat-user-avatar">我</Avatar></div>
      <div className="assistant-message"><div className="assistant-avatar"><ThunderboltFilled /></div><div className="assistant-body">
        <div className="assistant-name">Boke Oncall <span>· 工单助手</span></div>
        <p className="ticket-assistant-copy">{completed ? '这张工单已处理完成。下面是工单概况与处理记录。' : '已为你生成 Oncall 工单，协作群创建流程已模拟完成。'}</p>
        <Card className="ticket-summary-card" bordered={false}>
          <div className="ticket-card-heading"><div className="ticket-card-icon"><TeamOutlined /></div><div><span>ONCALL REQUEST</span><h2>{ticket.id}</h2></div><Tag className={completed ? 'status-tag completed' : 'status-tag'} bordered={false} icon={completed ? <CheckCircleFilled /> : <ClockCircleOutlined />}>{completed ? '已完成' : '进行中'}</Tag></div>
          <Descriptions column={1} size="small" colon={false} items={[
            { key: 'tenant', label: '目标租户', children: ticket.tenantName },
            { key: 'issue', label: '问题描述', children: ticket.description },
            { key: 'group', label: '协作群', children: ticket.groupName },
            { key: 'created', label: '创建时间', children: date },
          ]} />
          <div className="ticket-card-notice"><FileTextOutlined /> 当前为演示数据，尚未真正创建飞书群聊或通知值班人员。</div>
        </Card>
        <div className="ticket-timeline"><h3>处理进度</h3><Timeline items={[
          { color: 'green', children: <><strong>工单已创建</strong><span>{date}</span></> },
          { color: completed ? 'green' : 'blue', children: <><strong>{completed ? '问题已解决并结案' : '等待值班同学处理'}</strong><span>{completed ? '演示处理记录' : '接入真实服务后实时更新'}</span></> },
        ]} /></div>
      </div></div>
    </div>
  )
}
