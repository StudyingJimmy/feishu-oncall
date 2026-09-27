import { ArrowUpOutlined, CheckOutlined, LoadingOutlined, SearchOutlined } from '@ant-design/icons'
import { Avatar, Button, Input } from 'antd'
import type { KeyboardEvent } from 'react'
import type { TenantMatch } from '../domain/types'

interface Props {
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  submitting: boolean
  matches: TenantMatch[]
  rankingLoading: boolean
  onSelectTenant: (match: TenantMatch) => void
}

export default function MobileComposer({ value, onChange, onSubmit, submitting, matches, rankingLoading, onSelectTenant }: Props) {
  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault()
      if (value.trim() && !submitting) onSubmit()
    }
  }

  const chooseTenant = (match: TenantMatch) => {
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
    onSelectTenant(match)
  }

  return (
    <div className="mobile-composer-area">
      {value.trim() && (
        <section className="mobile-live-ranking" aria-label="实时租户匹配">
          <div className="mobile-live-header"><div><span className="mobile-live-dot" />实时租户匹配</div><span>{rankingLoading ? <><LoadingOutlined spin /> 匹配中</> : '点击租户发起 Oncall'}</span></div>
          <div className="mobile-live-list">
            {matches.map((match, index) => (
              <button type="button" key={match.tenant.id} className="mobile-live-row" onClick={() => chooseTenant(match)}>
                <span className="mobile-live-rank">{String(index + 1).padStart(2, '0')}</span>
                <Avatar style={{ background: match.tenant.color }}>{match.tenant.avatar}</Avatar>
                <span className="mobile-live-name"><strong>{match.tenant.name}</strong><small>{match.matchedTerms.length ? `匹配 ${match.matchedTerms.slice(0, 2).join('、')}` : match.tenant.domain}</small></span>
                <CheckOutlined className="mobile-live-check" />
              </button>
            ))}
          </div>
        </section>
      )}
      <div className="mobile-composer-box">
        <div className="mobile-composer-input-row">
          <Input.TextArea
            value={value}
            onChange={(event) => onChange(event.target.value)}
            onKeyDown={handleKeyDown}
            autoSize={{ minRows: 1, maxRows: 4 }}
            placeholder="描述问题，搜索已有方案…"
            aria-label="输入问题"
          />
          <Button type="primary" shape="circle" icon={<ArrowUpOutlined />} onClick={onSubmit} disabled={!value.trim()} loading={submitting} aria-label="搜索问题" />
        </div>
        <div className="mobile-composer-hint"><SearchOutlined /> 回车搜索问题 <span>·</span> 点击匹配租户发起 Oncall</div>
      </div>
    </div>
  )
}
