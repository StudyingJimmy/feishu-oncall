import { ArrowUpOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Input, Tooltip } from 'antd'
import type { KeyboardEvent } from 'react'
import type { TenantMatch } from '../domain/types'
import TenantRanking from './TenantRanking'

interface Props {
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  submitting: boolean
  matches: TenantMatch[]
  rankingLoading: boolean
  onTenantSelect: (match: TenantMatch) => void
  placement: 'home' | 'conversation'
}

export default function QuestionComposer({ value, onChange, onSubmit, submitting, matches, rankingLoading, onTenantSelect, placement }: Props) {
  const ranking = (
    <TenantRanking query={value} matches={matches} loading={rankingLoading} onSelect={onTenantSelect} compact={placement === 'conversation'} />
  )

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault()
      if (value.trim() && !submitting) onSubmit()
    }
  }

  return (
    <div className={`composer-section ${placement}`}>
      {placement === 'conversation' && value.trim() && ranking}
      <div className="composer-box">
        <Input.TextArea
          className="composer-input"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          onKeyDown={handleKeyDown}
          autoSize={{ minRows: 2, maxRows: 5 }}
          placeholder="描述你遇到的问题，或搜索已有解决方案…"
          aria-label="输入问题"
        />
        <div className="composer-toolbar">
          <span className="composer-mode"><SearchOutlined /> 问题搜索 <span className="toolbar-separator" /> 输入时同步推荐租户</span>
          <Tooltip title="按 Enter 搜索，Shift + Enter 换行">
            <Button
              className="composer-submit"
              type="primary"
              icon={<ArrowUpOutlined />}
              onClick={onSubmit}
              disabled={!value.trim()}
              loading={submitting}
              aria-label="搜索问题"
            >
              搜索
            </Button>
          </Tooltip>
        </div>
      </div>
      {placement === 'home' && ranking}
      <div className="composer-disclaimer">演示环境 · 搜索与租户匹配来自模拟数据</div>
    </div>
  )
}
