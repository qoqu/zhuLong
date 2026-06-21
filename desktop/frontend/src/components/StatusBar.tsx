import type { Language, RuntimeStats } from '../types'
import { useT } from '../i18n'
import type { AgentStatus } from '../types'

interface StatusBarProps {
  language: Language
  status: AgentStatus
  model: string
  stats: RuntimeStats
  darkMode: boolean
  onToggleDarkMode: () => void
  languageToggle: () => void
}

const statusColor: Record<AgentStatus, string> = {
  idle: 'var(--fg-faint)',
  planning: 'var(--warn)',
  executing: 'var(--ok)',
  reflecting: 'var(--accent)',
  replanning: 'var(--warn)',
  waiting_human: 'var(--warn)',
  done: 'var(--ok)',
  error: 'var(--err)',
}

export function StatusBar(props: StatusBarProps) {
  const t = useT(props.language)
  const label: Record<AgentStatus, string> = {
    idle: t.statusIdle,
    planning: t.statusPlanning,
    executing: t.statusExecuting,
    reflecting: t.statusReflecting,
    replanning: t.statusReplanning,
    waiting_human: t.statusWaiting,
    done: t.statusDone,
    error: t.statusError,
  }

  return (
    <footer className="statusbar">
      <div className="statusbar__group">
        <span className="statusbar__dot" style={{ background: statusColor[props.status] }} />
        <span>{label[props.status]}</span>
      </div>
      <span className="statusbar__sep">·</span>
      <div className="statusbar__group">{props.model}</div>
      <span className="statusbar__sep">·</span>
      <div className="statusbar__group">
        {t.hitRate} {(props.stats.cacheHitRatio * 100).toFixed(0)}%
      </div>
      <span className="statusbar__sep">·</span>
      <div className="statusbar__group">
        {t.sessionTokens} {props.stats.sessionTokens.toLocaleString()}
      </div>
      <span className="statusbar__sep">·</span>
      <div className="statusbar__group">
        {props.stats.balance}
      </div>
      <div className="statusbar__spacer" />
      <button
        className="statusbar__icon"
        onClick={props.onToggleDarkMode}
        title={props.darkMode ? 'Light' : 'Dark'}
      >
        {props.darkMode ? '☀' : '☾'}
      </button>
      <button className="statusbar__icon" onClick={props.languageToggle} title="Language">
        {props.language === 'en' ? '中' : 'EN'}
      </button>
    </footer>
  )
}
