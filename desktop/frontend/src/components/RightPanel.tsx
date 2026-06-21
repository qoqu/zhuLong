import type { RightPanelTab, RuntimeStats, Language, FileChange } from '../types'
import { useT } from '../i18n'

interface RightPanelProps {
  language: Language
  tab: RightPanelTab
  onChangeTab: (t: RightPanelTab) => void
  stats: RuntimeStats
  files: string[]
  changes: FileChange[]
}

export function RightPanel(props: RightPanelProps) {
  const t = useT(props.language)
  return (
    <aside className="right-panel">
      <div className="right-panel__tabs">
        <button
          className={`right-panel__tab ${props.tab === 'overview' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('overview')}
        >
          <span className="right-panel__tab-icon">≣</span>
          {t.overview}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'files' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('files')}
        >
          <span className="right-panel__tab-icon">📄</span>
          {t.files}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'changes' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('changes')}
        >
          <span className="right-panel__tab-icon">↻</span>
          {t.changes}
        </button>
      </div>

      <div className="right-panel__content">
        {props.tab === 'overview' && <OverviewTab language={props.language} stats={props.stats} />}
        {props.tab === 'files' && <FilesTab language={props.language} files={props.files} />}
        {props.tab === 'changes' && <ChangesTab language={props.language} changes={props.changes} />}
      </div>
    </aside>
  )
}

function OverviewTab(props: { language: Language; stats: RuntimeStats }) {
  const t = useT(props.language)
  const s = props.stats
  const fmt = (n: number) => n.toLocaleString()
  return (
    <div className="overview-tab">
      {/* Context window donut */}
      <section className="overview-section">
        <div className="overview-section__header">
          <h3>{t.contextWindow}</h3>
          <span className="overview-section__hint">{t.currentWindowUsage}</span>
        </div>
        <Donut percent={s.usagePercent} used={s.totalUsed} total={s.totalLimit} />
        <ul className="breakdown">
          <li>
            <span className="dot dot--prompt" />
            <span className="label">{t.prompt}</span>
            <span className="value">{fmt(s.prompt)}</span>
          </li>
          <li>
            <span className="dot dot--completion" />
            <span className="label">{t.completion}</span>
            <span className="value">{fmt(s.completion)}</span>
          </li>
          <li>
            <span className="dot dot--reasoning" />
            <span className="label">{t.reasoning}</span>
            <span className="value">{fmt(s.reasoning)}</span>
          </li>
          <li>
            <span className="dot dot--other" />
            <span className="label">{t.other}</span>
            <span className="value">{fmt(s.other)}</span>
          </li>
        </ul>
        <div className="total-line">
          <span>{t.total}</span>
          <span>
            {fmt(s.totalUsed)} / {fmt(s.totalLimit)}
          </span>
        </div>
      </section>

      {/* Runtime */}
      <section className="overview-section">
        <h3>{t.runtime}</h3>
        <div className="runtime-grid">
          <div className="runtime-cell">
            <div className="runtime-cell__label">{t.elapsed}</div>
            <div className="runtime-cell__value">{s.elapsed}</div>
          </div>
          <div className="runtime-cell">
            <div className="runtime-cell__label">{t.requests}</div>
            <div className="runtime-cell__value">{s.requestCount}</div>
          </div>
        </div>
        <div className="runtime-row">
          <span>{t.sessionTokens}</span>
          <span>{fmt(s.sessionTokens)}</span>
        </div>
        <div className="runtime-row">
          <span>{t.cacheBalance}</span>
          <span>{(s.cacheHitRatio * 100).toFixed(0)}%</span>
        </div>
      </section>

      {/* Cost */}
      <section className="overview-section">
        <h3>{t.cost}</h3>
        <div className="cost-grid">
          <div className="cost-cell">
            <div className="cost-cell__label">{t.cacheBalance}</div>
            <div className="cost-cell__value">—</div>
          </div>
          <div className="cost-cell">
            <div className="cost-cell__label">{t.sessionCost}</div>
            <div className="cost-cell__value">{s.sessionCost}</div>
          </div>
        </div>
        <div className="cost-row">
          <span>{t.mainModel}</span>
          <span>
            ${s.mainCost.toFixed(4)}{' '}
            <span className="muted">/ {s.mainCount} 次</span>
          </span>
        </div>
        <div className="cost-row">
          <span>{t.subModel}</span>
          <span>
            ${s.subCost.toFixed(4)}{' '}
            <span className="muted">/ {s.subCount} 次</span>
          </span>
        </div>
      </section>
    </div>
  )
}

function Donut(props: { percent: number; used: number; total: number }) {
  const size = 160
  const stroke = 12
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const offset = c * (1 - Math.min(props.percent, 1))
  return (
    <div className="donut">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle cx={size / 2} cy={size / 2} r={r} stroke="var(--bg-soft)" strokeWidth={stroke} fill="none" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke="var(--accent)"
          strokeWidth={stroke}
          fill="none"
          strokeDasharray={c}
          strokeDashoffset={offset}
          strokeLinecap="round"
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
        />
        <text x="50%" y="48%" textAnchor="middle" className="donut__big">
          {props.used >= 1000 ? Math.round(props.used / 1000) + 'k' : props.used}
        </text>
        <text x="50%" y="62%" textAnchor="middle" className="donut__small">
          / {Math.round(props.total / 1000)}k tokens
        </text>
      </svg>
      <div className="donut__percent">{props.percent.toFixed(1)}%</div>
    </div>
  )
}

function FilesTab(props: { language: Language; files: string[] }) {
  const t = useT(props.language)
  if (props.files.length === 0) {
    return (
      <div className="panel-placeholder">
        <p>📄</p>
        <p>{t.noFiles}</p>
      </div>
    )
  }
  return (
    <ul className="file-list">
      {props.files.map((f, i) => (
        <li key={i} className="file-list__item">
          📄 {f}
        </li>
      ))}
    </ul>
  )
}

function ChangesTab(props: { language: Language; changes: FileChange[] }) {
  const t = useT(props.language)
  if (props.changes.length === 0) {
    return (
      <div className="panel-placeholder">
        <p>↻</p>
        <p>{t.noChanges}</p>
      </div>
    )
  }
  return (
    <ul className="change-list">
      {props.changes.map((c, i) => (
        <li key={i} className={`change-list__item change-list__item--${c.kind}`}>
          <span className="change-list__kind">{c.kind === 'created' ? '+' : c.kind === 'deleted' ? '−' : '✎'}</span>
          <span className="change-list__path">{c.path}</span>
          <span className="change-list__time">{c.time}</span>
        </li>
      ))}
    </ul>
  )
}
