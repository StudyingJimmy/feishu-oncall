import { ArrowRightOutlined, BookOutlined, CheckCircleOutlined, ClockCircleOutlined, FileTextOutlined, LoadingOutlined, ThunderboltFilled } from '@ant-design/icons'
import { Avatar, Card, Empty, Tag } from 'antd'
import type { SearchTurn, TenantMatch } from '../domain/types'

interface Props {
  turns: SearchTurn[]
  loading: boolean
  pendingQuestion: string
  onSelectTenant: (match: TenantMatch, question: string) => void
}

function SearchAnswer({ turn, onSelectTenant }: { turn: SearchTurn; onSelectTenant: Props['onSelectTenant'] }) {
  return (
    <div className="answer-content">
      <p className="answer-intro">我查到了与你的问题相关的处理资料。可以先参考这些内容；如果仍需人工协助，选择下方租户发起 Oncall。</p>
      <div className="answer-section-title"><BookOutlined /> 相关处理资料 <span>{turn.hits.length} 条结果</span></div>
      {turn.hits.length ? <div className="knowledge-list">
        {turn.hits.map((hit, index) => (
          <Card className="knowledge-card" key={hit.id} size="small" hoverable>
            <div className="knowledge-meta"><span className="knowledge-index">{String(index + 1).padStart(2, '0')}</span><Tag className="source-tag" bordered={false} icon={hit.source === '知识库' ? <BookOutlined /> : <FileTextOutlined />}>{hit.source}</Tag><span className="knowledge-updated"><ClockCircleOutlined /> {hit.updatedAt}</span></div>
            <strong>{hit.title}</strong>
            <p>{hit.summary}</p>
          </Card>
        ))}
      </div> : <div className="knowledge-empty"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="没有找到直接匹配的资料，可联系推荐租户继续处理" /></div>}
      <div className="answer-section-title tenants-title"><ThunderboltFilled /> 推荐联系 <span>按匹配度排序</span></div>
      <div className="answer-tenant-list">
        {turn.tenantMatches.slice(0, 2).map((match) => (
          <button key={match.tenant.id} className="answer-tenant" type="button" onClick={() => onSelectTenant(match, turn.question)}>
            <Avatar style={{ background: match.tenant.color }}>{match.tenant.avatar}</Avatar>
            <span><strong>{match.tenant.name}</strong><small>{match.tenant.domain}</small></span>
            <span className="answer-tenant-action">发起 Oncall <ArrowRightOutlined /></span>
          </button>
        ))}
      </div>
      <div className="answer-footnote"><CheckCircleOutlined /> 以上搜索内容仅用于界面演示，后续由真实知识检索接口返回。</div>
    </div>
  )
}

export default function ConversationView({ turns, loading, pendingQuestion, onSelectTenant }: Props) {
  return (
    <div className="conversation-stream" aria-live="polite">
      {turns.map((turn) => (
        <div className="conversation-turn" key={turn.id}>
          <div className="user-message"><span>{turn.question}</span><Avatar className="chat-user-avatar">我</Avatar></div>
          <div className="assistant-message"><div className="assistant-avatar"><ThunderboltFilled /></div><div className="assistant-body"><div className="assistant-name">Boke Oncall <span>· 搜索助手</span></div><SearchAnswer turn={turn} onSelectTenant={onSelectTenant} /></div></div>
        </div>
      ))}
      {loading && <div className="conversation-turn"><div className="user-message"><span>{pendingQuestion}</span><Avatar className="chat-user-avatar">我</Avatar></div><div className="assistant-message searching-state"><div className="assistant-avatar"><ThunderboltFilled /></div><div className="assistant-body"><div className="assistant-name">Boke Oncall</div><p><LoadingOutlined spin /> 正在检索知识库并分析租户…</p></div></div></div>}
    </div>
  )
}
