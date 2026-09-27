import { ArrowRightOutlined, CheckOutlined, LoadingOutlined } from '@ant-design/icons'
import { Avatar, Empty } from 'antd'
import type { TenantMatch } from '../domain/types'

interface Props {
  query: string
  matches: TenantMatch[]
  loading: boolean
  onSelect: (match: TenantMatch) => void
  compact?: boolean
}

export default function TenantRanking({ query, matches, loading, onSelect, compact = false }: Props) {
  return (
    <section className={`ranking-panel ${compact ? 'is-compact' : ''}`} aria-label="租户匹配排行">
      <div className="ranking-heading">
        <div><span className="eyebrow-dot" /><strong>{query.trim() ? '实时租户匹配' : '常见服务租户'}</strong><p>{query.trim() ? '根据输入动态排序 · 选择租户可发起 Oncall' : '开始描述问题，推荐顺序会实时更新'}</p></div>
        <span className="ranking-live">{loading ? <LoadingOutlined spin /> : <span className="live-dot" />}{loading ? '匹配中' : '实时更新'}</span>
      </div>
      {matches.length ? (
        <div className="ranking-list">
          {matches.map((match, index) => (
            <button className="tenant-row" key={match.tenant.id} type="button" onClick={() => onSelect(match)}>
              <span className={`tenant-rank rank-${index + 1}`}>{String(index + 1).padStart(2, '0')}</span>
              <Avatar className="tenant-avatar" style={{ background: match.tenant.color }}>{match.tenant.avatar}</Avatar>
              <span className="tenant-main"><strong>{match.tenant.name}</strong><span>{match.tenant.domain} <span className="tenant-divider">/</span> {match.tenant.summary}</span></span>
              <span className="tenant-match-reason">{match.matchedTerms.length ? <><CheckOutlined /> 命中 {match.matchedTerms.slice(0, 2).join('、')}</> : '查看详情'}</span>
              <ArrowRightOutlined className="tenant-arrow" />
            </button>
          ))}
        </div>
      ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无租户" />}
      <div className="ranking-footer">租户数据为演示内容，实际匹配逻辑可接入租户服务与知识库。</div>
    </section>
  )
}
