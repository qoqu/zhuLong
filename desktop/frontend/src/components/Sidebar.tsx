import type { Language, AgentInfo, ProjectInfo, SidebarView } from '../types'
import { useT } from '../i18n'

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
          <div className="sidebar__placeholder">{t.agentsView}</div>
        )}
        {props.view === 'history' && (
          <div className="sidebar__placeholder">{t.historyView}</div>
        )}
        {props.view === 'trash' && (
          <div className="sidebar__placeholder">{t.trashView}</div>
        )}
        {props.view === 'settings' && (
          <div className="sidebar__placeholder">{t.settingsView}</div>
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
      {props.projects.flatMap((p) => p.sessions).map((s) => (
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
