import type { Language, AgentStatus, RuntimeStats, ExecutionMode } from '../types'
import { useT, resolveLanguage } from '../i18n'

interface StatusBarProps {
  language: Language
  status: AgentStatus
  model: string
  stats: RuntimeStats
  executionMode: ExecutionMode
}

const statusLabel = (l: Language, s: AgentStatus) => {
  if (l === 'en') {
    return s === 'waiting_human' ? 'Waiting'
      : s === 'idle' ? 'Ready'
      : s === 'done' ? 'Done'
      : s === 'error' ? 'Error'
      : s.charAt(0).toUpperCase() + s.slice(1)
  }
  return s === 'waiting_human' ? '等待'
    : s === 'idle' ? '就绪'
    : s === 'done' ? '完成'
    : s === 'error' ? '错误'
    : s === 'planning' ? '规划中'
    : s === 'executing' ? '执行中'
    : s === 'reflecting' ? '反省中'
    : s === 'replanning' ? '重规划中'
    : s
}

const statusColor = (s: AgentStatus) => {
  switch (s) {
    case 'idle': return 'var(--fg-faint)'
    case 'done': return 'var(--ok)'
    case 'error': return 'var(--err)'
    case 'waiting_human': return 'var(--warn)'
    default: return 'var(--accent)'
  }
}

export function StatusBar(props: StatusBarProps) {
  const t = useT(props.language)
  const s = props.stats
  const isZh = resolveLanguage(props.language) === 'zh'
  const modeLabel = props.executionMode === 'ask' ? (isZh ? '询问' : 'Ask')
    : props.executionMode === 'auto' ? (isZh ? '自动' : 'Auto')
    : (isZh ? 'YOLO' : 'YOLO')
  const modeColor = props.executionMode === 'ask' ? 'var(--ok)' : props.executionMode === 'auto' ? 'var(--warn)' : 'var(--err)'
  return (
    <div className="statusbar">
      <div className="statusbar__group">
        <span className="statusbar__dot" style={{ background: statusColor(props.status) }} />
        <span>{props.model}</span>
      </div>
      <span className="statusbar__badge" style={{ color: modeColor, borderColor: modeColor }}>{modeLabel}</span>
      <span className="statusbar__sep">·</span>
      <span>{t.status || 'Status'}: <strong>{statusLabel(resolveLanguage(props.language), props.status)}</strong></span>
      {s.fsmState && (
        <>
          <span className="statusbar__sep">·</span>
          <span>FSM: <strong>{s.fsmState}</strong></span>
        </>
      )}
      <span className="statusbar__sep">·</span>
      <span>{t.thisHit}: <strong>{s.cacheHit}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.avgHit}: <strong>{s.avgHit}</strong></span>
      {s.budgetUsed !== undefined && s.budgetLimit && s.budgetLimit > 0 && (
        <>
          <span className="statusbar__sep">·</span>
          <span>{t.budget || 'Budget'}: <strong style={{ color: s.budgetWarning ? 'var(--warn)' : 'var(--ok)' }}>{Math.round((s.budgetUsed / s.budgetLimit) * 100)}%</strong></span>
        </>
      )}
      <span className="statusbar__sep">·</span>
      <span>{t.sessionTokens}: <strong>{s.sessionTokens.toLocaleString()}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.thisTokens}: <strong>{s.thisTokens.toLocaleString()}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.thisFee}: <strong>{s.thisFee}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.currentSession}: <strong>{s.currentSession}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.contextUsed}: <strong>{s.contextUsed.toLocaleString()}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.compressThreshold}: <strong>{s.compressPct}%</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.balance}: <strong style={{ color: 'var(--ok)' }}>{s.balance}</strong></span>
      <span className="statusbar__sep">·</span>
      <span>{t.sessionCost}: <strong>{s.sessionCost}</strong></span>
      <span className="statusbar__spacer" />
      <span className="statusbar__group">
        <span
          className="statusbar__dot"
          style={{ background: statusColor(props.status) }}
        />
        <span>{statusLabel(resolveLanguage(props.language), props.status)}</span>
      </span>
    </div>
  )
}
