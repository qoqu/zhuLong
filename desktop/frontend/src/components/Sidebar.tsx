import { useState, useEffect } from 'react'
import type { Language, AgentInfo, ProjectInfo, SidebarView } from '../types'
import { useT } from '../i18n'

interface AppConfig {
  monitorPath: string
  backupMode: string
  backupOnFail: boolean
  dashboardPort: number
  pluginsPath: string
}

interface SidebarProps {
  language: Language
  onLanguageChange: (l: Language) => void
  collapsed: boolean
  onToggleCollapsed: () => void
  view: SidebarView
  onChangeView: (v: SidebarView) => void
  agents: AgentInfo[]
  projects: ProjectInfo[]
  activeAgentId: string
  onSelectAgent: (id: string) => void
  activeSessionId: string
  onSelectSession: (id: string) => void
  onNewSession: () => void
  onDeleteSession: (id: string) => void
}

export function Sidebar(props: SidebarProps) {
  const t = useT(props.language)
  if (props.collapsed) {
    return (
      <aside className="sidebar sidebar--collapsed">
        <button className="sidebar__expand-btn" onClick={props.onToggleCollapsed} title="Expand">
          →
        </button>
        <div className="sidebar__collapsed-stack">
          {props.agents.map((a) => (
            <button
              key={a.id}
              className="sidebar__icon-btn"
              title={a.name}
              onClick={() => props.onSelectAgent(a.id)}
            >
              {a.name[0]}
            </button>
          ))}
          <button className="sidebar__icon-btn" title={t.newSession} onClick={props.onNewSession}>
            ✚
          </button>
        </div>
      </aside>
    )
  }

  return (
    <aside className="sidebar">
      {/* Brand + collapse */}
      <div className="sidebar__brand">
        <span className="sidebar__brand-name">{t.brand}</span>
        <button className="sidebar__icon-btn" onClick={props.onToggleCollapsed} title="Collapse">
          ←
        </button>
      </div>

      {/* New session */}
      <button className="btn-new-session" onClick={props.onNewSession}>
        <span>✎</span>
        <span>{t.newSession}</span>
      </button>

      {/* Search */}
      <div className="sidebar__search">
        <span>⌕</span>
        <input placeholder={t.searchSessions} />
      </div>

      {/* Agent tabs */}
      <div className="agent-tabs">
        {props.agents.map((a) => (
          <button
            key={a.id}
            className={`agent-tab ${props.activeAgentId === a.id ? 'active' : ''}`}
            onClick={() => props.onSelectAgent(a.id)}
            title={a.name}
          >
            <span className="agent-tab__name">{a.name}</span>
            {a.yolo && <span className="agent-tab__badge">YOLO</span>}
          </button>
        ))}
        <button className="agent-tab agent-tab--add" title={t.addAgent}>+</button>
        <button className="agent-tab agent-tab--search" title={t.searchAgents}>⌕</button>
      </div>

      <div className="sidebar__body">
        {props.view === 'projects' && (
          <ProjectTree
            language={props.language}
            projects={props.projects}
            activeSessionId={props.activeSessionId}
            onSelectSession={props.onSelectSession}
            onDeleteSession={props.onDeleteSession}
          />
        )}
        {props.view === 'agents' && (
          <div className="agents-view">
            <h3 className="agents-view__title">{t.agents}</h3>
            <div className="agents-view__list">
              {props.agents.map((agent) => (
                <div 
                  key={agent.id} 
                  className={`agents-view__item ${props.activeAgentId === agent.id ? 'active' : ''}`}
                  onClick={() => props.onSelectAgent(agent.id)}
                >
                  <div className="agents-view__avatar">{agent.name[0]}</div>
                  <div className="agents-view__info">
                    <div className="agents-view__name">{agent.name}</div>
                    <div className="agents-view__model">{agent.model}</div>
                  </div>
                  {agent.yolo && <span className="agents-view__badge">YOLO</span>}
                </div>
              ))}
            </div>
          </div>
        )}
        {props.view === 'history' && (
          <div className="history-view">
            <h3 className="history-view__title">{t.history}</h3>
            <div className="history-view__list">
              {props.projects.flatMap((p) => p.sessions).slice(0, 10).map((session) => (
                <div 
                  key={session.id}
                  className={`history-view__item ${props.activeSessionId === session.id ? 'active' : ''}`}
                  onClick={() => props.onSelectSession(session.id)}
                >
                  <div className="history-view__title-text">{session.title}</div>
                  <div className="history-view__meta">
                    {session.messageCount} {props.language === 'zh' ? '轮' : 'msgs'} · {session.updatedAt}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
        {props.view === 'trash' && (
          <div className="trash-view">
            <h3 className="trash-view__title">{t.trash}</h3>
            <div className="trash-view__empty">
              <p>🗑</p>
              <p>{props.language === 'zh' ? '回收站为空' : 'Trash is empty'}</p>
            </div>
          </div>
        )}
        {props.view === 'settings' && (
          <div className="settings-view">
            <h3 className="settings-view__title">{t.settings}</h3>

            <div className="settings-view__section">
              <h4>{props.language === 'zh' ? '外观' : 'Appearance'}</h4>
              <div className="settings-view__item">
                <span>{props.language === 'zh' ? '主题' : 'Theme'}</span>
                <button
                  className="pill"
                  onClick={() => props.onLanguageChange(props.language === 'zh' ? 'en' : 'zh')}
                >
                  {props.language === 'zh' ? '深色' : 'Dark'}
                </button>
              </div>
            </div>

            <div className="settings-view__section">
              <h4>{props.language === 'zh' ? '语言' : 'Language'}</h4>
              <div className="settings-view__item">
                <span>{props.language === 'zh' ? '当前语言' : 'Current Language'}</span>
                <button
                  className="pill"
                  onClick={() => props.onLanguageChange(props.language === 'zh' ? 'en' : 'zh')}
                >
                  {props.language === 'zh' ? '中文' : 'English'}
                </button>
              </div>
            </div>

            <SettingsPanel language={props.language} />

            <div className="settings-view__section">
              <h4>{props.language === 'zh' ? '关于' : 'About'}</h4>
              <div className="settings-view__item">
                <span>{props.language === 'zh' ? '版本' : 'Version'}</span>
                <span>v0.4.0</span>
              </div>
              <div className="settings-view__item">
                <span>{props.language === 'zh' ? '模型' : 'Model'}</span>
                <span>DeepSeek</span>
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="sidebar__footer">
        <NavItem icon="🕐" label={t.history} active={props.view === 'history'} onClick={() => props.onChangeView('history')} />
        <NavItem icon="🗑" label={t.trash} active={props.view === 'trash'} onClick={() => props.onChangeView('trash')} />
        <NavItem icon="⚙" label={t.settings} active={props.view === 'settings'} onClick={() => props.onChangeView('settings')} />
        <NavItem
          icon={props.language === 'zh' ? '中' : 'EN'}
          label={t.language}
          active={false}
          onClick={() => props.onLanguageChange(props.language === 'zh' ? 'en' : 'zh')}
        />
      </div>
    </aside>
  )
}

function NavItem(props: { icon: string; label: string; active: boolean; onClick: () => void }) {
  return (
    <button className={`sidebar__nav ${props.active ? 'active' : ''}`} onClick={props.onClick}>
      <span className="sidebar__nav-icon">{props.icon}</span>
      <span>{props.label}</span>
    </button>
  )
}

function ProjectTree(props: {
  language: Language
  projects: ProjectInfo[]
  activeSessionId: string
  onSelectSession: (id: string) => void
  onDeleteSession: (id: string) => void
}) {
  return (
    <div className="project-tree">
      <div className="project-tree__header">
        <span>{props.projects[0]?.name || '项目'}</span>
        <div className="project-tree__actions">
          <button title="Sort">⇅</button>
          <button title="Layout">▦</button>
        </div>
      </div>
      {props.projects.flatMap((p) => p.sessions || []).filter(Boolean).map((s) => (
        <button
          key={s.id}
          className={`session-item ${s.id === props.activeSessionId ? 'active' : ''}`}
          onClick={() => props.onSelectSession(s.id)}
          onContextMenu={(e) => {
            e.preventDefault()
            if (window.confirm(props.language === 'zh' ? `删除「${s.title}」？` : `Delete "${s.title}"?`)) {
              props.onDeleteSession(s.id)
            }
          }}
          title={s.title}
        >
          <div className="session-item__title">{s.title}</div>
          <div className="session-item__meta">
            {s.messageCount} {props.language === 'zh' ? '轮' : 'msgs'} · {s.toolCount} {props.language === 'zh' ? '工具' : 'tools'} · {s.updatedAt}
          </div>
        </button>
      ))}
    </div>
  )
}

// SettingsPanel - 第四步: 应用配置面板
function SettingsPanel(props: { language: Language }) {
  const isZh = props.language === 'zh'
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [dashRunning, setDashRunning] = useState(false)
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')

  useEffect(() => {
    const backend = (window as any).go?.main?.App
    if (!backend) return
    backend.GetConfig().then((c: AppConfig) => setConfig(c))
  }, [])

  const callBackend = async (fn: () => Promise<any>, okMsg: string) => {
    const backend = (window as any).go?.main?.App
    if (!backend) return
    setBusy(true)
    try {
      await fn()
      setMsg(okMsg)
      setTimeout(() => setMsg(''), 2000)
    } catch (e: any) {
      setMsg('Error: ' + (e?.message || String(e)))
    } finally {
      setBusy(false)
    }
  }

  const handleBackupMode = (mode: string) => {
    if (!config) return
    setConfig({ ...config, backupMode: mode })
    const backend = (window as any).go?.main?.App
    backend?.SetBackupMode(mode, config.backupOnFail)
  }

  const handleBackupOnFail = (onFail: boolean) => {
    if (!config) return
    setConfig({ ...config, backupOnFail: onFail })
    const backend = (window as any).go?.main?.App
    backend?.SetBackupMode(config.backupMode, onFail)
  }

  const handleMonitorPath = () => {
    if (!config) return
    const next = window.prompt(isZh ? '监视路径' : 'Monitor path', config.monitorPath)
    if (next && next.trim()) {
      setConfig({ ...config, monitorPath: next.trim() })
      const backend = (window as any).go?.main?.App
      backend?.SetMonitorPath(next.trim())
    }
  }

  const handleDashboardPort = () => {
    if (!config) return
    const next = window.prompt(isZh ? 'Dashboard 端口' : 'Dashboard port', String(config.dashboardPort))
    const port = parseInt(next || '', 10)
    if (port > 0 && port < 65536) {
      setConfig({ ...config, dashboardPort: port })
      const backend = (window as any).go?.main?.App
      backend?.SetDashboardPort(port)
    }
  }

  const handleStartDash = async () => {
    const backend = (window as any).go?.main?.App
    if (!backend) return
    try {
      await backend.StartDashboard()
      setDashRunning(true)
      setMsg(isZh ? `Dashboard 已启动 :${config?.dashboardPort}` : `Dashboard started on :${config?.dashboardPort}`)
      setTimeout(() => setMsg(''), 3000)
    } catch (e: any) {
      setMsg('Error: ' + (e?.message || String(e)))
    }
  }

  const handleStopDash = async () => {
    const backend = (window as any).go?.main?.App
    if (!backend) return
    try {
      await backend.StopDashboard()
      setDashRunning(false)
      setMsg(isZh ? 'Dashboard 已停止' : 'Dashboard stopped')
      setTimeout(() => setMsg(''), 2000)
    } catch (e: any) {
      setMsg('Error: ' + (e?.message || String(e)))
    }
  }

  const handleTriggerBackup = () => {
    const backend = (window as any).go?.main?.App
    callBackend(() => backend?.TriggerBackup(), isZh ? '✓ 备份已创建' : '✓ Backup created')
  }

  if (!config) {
    return (
      <div className="settings-view__section">
        <h4>{isZh ? '加载中...' : 'Loading...'}</h4>
      </div>
    )
  }

  return (
    <>
      {msg && (
        <div style={{
          padding: '6px 10px',
          background: msg.startsWith('Error') ? 'var(--err)' : 'var(--ok)',
          color: 'white',
          borderRadius: 'var(--radius)',
          fontSize: 'var(--text-xs)',
          marginBottom: 'var(--space-2)',
        }}>
          {msg}
        </div>
      )}

      <div className="settings-view__section">
        <h4>{isZh ? '环境监控' : 'Environment Monitor'}</h4>
        <div className="settings-view__item">
          <span>{isZh ? '监视路径' : 'Monitor Path'}</span>
          <button className="pill" onClick={handleMonitorPath} disabled={busy}>
            {config.monitorPath}
          </button>
        </div>
      </div>

      <div className="settings-view__section">
        <h4>{isZh ? '自动备份' : 'Auto Backup'}</h4>
        <div className="settings-view__item">
          <span>{isZh ? '触发模式' : 'Mode'}</span>
          <div style={{ display: 'flex', gap: 4 }}>
            {[
              { v: 'off', z: '关闭', e: 'Off' },
              { v: 'immediate', z: '每步', e: 'Every Step' },
              { v: 'on-completion', z: '完成时', e: 'On Completion' },
            ].map(m => (
              <button
                key={m.v}
                className={`pill ${config.backupMode === m.v ? 'active' : ''}`}
                onClick={() => handleBackupMode(m.v)}
                disabled={busy}
              >
                {isZh ? m.z : m.e}
              </button>
            ))}
          </div>
        </div>
        <div className="settings-view__item">
          <span>{isZh ? '失败时也备份' : 'Backup on failure'}</span>
          <button
            className={`pill ${config.backupOnFail ? 'active' : ''}`}
            onClick={() => handleBackupOnFail(!config.backupOnFail)}
            disabled={busy}
          >
            {config.backupOnFail ? (isZh ? '是' : 'Yes') : (isZh ? '否' : 'No')}
          </button>
        </div>
        <div className="settings-view__item">
          <span>{isZh ? '手动备份' : 'Manual Backup'}</span>
          <button className="pill" onClick={handleTriggerBackup} disabled={busy}>
            {isZh ? '立即备份' : 'Backup Now'}
          </button>
        </div>
      </div>

      <div className="settings-view__section">
        <h4>{isZh ? 'Web Dashboard' : 'Web Dashboard'}</h4>
        <div className="settings-view__item">
          <span>{isZh ? '端口' : 'Port'}</span>
          <button className="pill" onClick={handleDashboardPort} disabled={busy}>
            :{config.dashboardPort}
          </button>
        </div>
        <div className="settings-view__item">
          <span>{isZh ? '状态' : 'Status'}</span>
          <button
            className={`pill ${dashRunning ? 'active' : ''}`}
            onClick={dashRunning ? handleStopDash : handleStartDash}
            disabled={busy}
          >
            {dashRunning ? (isZh ? '运行中' : 'Running') : (isZh ? '已停止' : 'Stopped')}
          </button>
        </div>
      </div>
    </>
  )
}
