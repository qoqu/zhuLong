import { useState } from 'react'
import { useT } from '../i18n'
import type { Language, SidebarView, Agent, Session } from '../types'
import { mockAgents, mockProjects } from '../data/mock'

interface SidebarProps {
  language: Language
  onLanguageChange: (l: Language) => void
  collapsed: boolean
  onToggleCollapsed: () => void
  view: SidebarView
  onChangeView: (v: SidebarView) => void
  activeAgentId: string
  onSelectAgent: (id: string) => void
  activeSessionId: string
  onSelectSession: (id: string) => void
  onNewSession: () => void
  onNewAgent: () => void
  onOpenCommandPalette: () => void
}

export function Sidebar(props: SidebarProps) {
  const t = useT(props.language)
  const [search, setSearch] = useState('')

  if (props.collapsed) {
    return (
      <aside className="sidebar sidebar--collapsed">
        <button
          className="sidebar__expand-btn"
          onClick={props.onToggleCollapsed}
          title="Expand sidebar"
        >
          <span className="icon-side">▸</span>
        </button>
        <div className="sidebar__collapsed-stack">
          <button
            className="sidebar__icon-btn"
            onClick={() => props.onChangeView('projects')}
            title={t.project}
          >
            📁
          </button>
          <button
            className="sidebar__icon-btn"
            onClick={() => props.onChangeView('robots')}
            title={t.robots}
          >
            🤖
          </button>
          <button
            className="sidebar__icon-btn"
            onClick={() => props.onChangeView('history')}
            title={t.history}
          >
            🕘
          </button>
          <button
            className="sidebar__icon-btn"
            onClick={() => props.onChangeView('recycle')}
            title={t.recycle}
          >
            🗑
          </button>
          <button
            className="sidebar__icon-btn"
            onClick={() => props.onChangeView('settings')}
            title={t.settings}
          >
            ⚙
          </button>
        </div>
      </aside>
    )
  }

  return (
    <aside className="sidebar">
      {/* Header: app name + collapse */}
      <div className="sidebar__brand">
        <div className="sidebar__brand-name">{t.appName}</div>
        <button
          className="sidebar__icon-btn sidebar__icon-btn--small"
          onClick={props.onToggleCollapsed}
          title="Collapse sidebar"
        >
          <span className="icon-side">◂</span>
        </button>
      </div>

      {/* New session button */}
      <div className="sidebar__new-session">
        <button className="btn-new-session" onClick={props.onNewSession}>
          <span>✎</span>
          <span>{t.newSession}</span>
        </button>
      </div>

      {/* Search */}
      <div className="sidebar__search">
        <span className="sidebar__search-icon">⌕</span>
        <input
          type="text"
          placeholder={t.searchProjectSession}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {/* View content */}
      <div className="sidebar__body">
        {props.view === 'projects' && (
          <ProjectsView
            language={props.language}
            search={search}
            activeAgentId={props.activeAgentId}
            onSelectAgent={props.onSelectAgent}
            activeSessionId={props.activeSessionId}
            onSelectSession={props.onSelectSession}
            onNewAgent={props.onNewAgent}
          />
        )}
        {props.view === 'robots' && <div className="sidebar__placeholder">{t.robots} (Coming soon)</div>}
        {props.view === 'history' && <div className="sidebar__placeholder">{t.history} (Coming soon)</div>}
        {props.view === 'recycle' && <div className="sidebar__placeholder">{t.recycle} (Coming soon)</div>}
        {props.view === 'settings' && <div className="sidebar__placeholder">{t.settings} (Coming soon)</div>}
      </div>

      {/* Footer navigation */}
      <div className="sidebar__footer">
        <button
          className={`sidebar__nav ${props.view === 'robots' ? 'active' : ''}`}
          onClick={() => props.onChangeView('robots')}
        >
          <span className="sidebar__nav-icon">🤖</span>
          <span>{t.robots}</span>
        </button>
        <button
          className={`sidebar__nav ${props.view === 'history' ? 'active' : ''}`}
          onClick={() => props.onChangeView('history')}
        >
          <span className="sidebar__nav-icon">🕘</span>
          <span>{t.history}</span>
        </button>
        <button
          className={`sidebar__nav ${props.view === 'recycle' ? 'active' : ''}`}
          onClick={() => props.onChangeView('recycle')}
        >
          <span className="sidebar__nav-icon">🗑</span>
          <span>{t.recycle}</span>
        </button>
        <button
          className={`sidebar__nav ${props.view === 'settings' ? 'active' : ''}`}
          onClick={() => props.onChangeView('settings')}
        >
          <span className="sidebar__nav-icon">⚙</span>
          <span>{t.settings}</span>
        </button>
      </div>
    </aside>
  )
}

function ProjectsView(props: {
  language: Language
  search: string
  activeAgentId: string
  onSelectAgent: (id: string) => void
  activeSessionId: string
  onSelectSession: (id: string) => void
  onNewAgent: () => void
}) {
  const t = useT(props.language)
  const [agentMenuOpen, setAgentMenuOpen] = useState(false)

  return (
    <>
      {/* Agent tabs */}
      <div className="agent-tabs">
        {mockAgents.map((a) => (
          <AgentTab
            key={a.id}
            agent={a}
            active={a.id === props.activeAgentId}
            onClick={() => props.onSelectAgent(a.id)}
          />
        ))}
        <button
          className="agent-tab agent-tab--add"
          onClick={() => setAgentMenuOpen((v) => !v)}
          title="Add agent"
        >
          +
        </button>
        <button className="agent-tab agent-tab--search" title="Search">
          ⌕
        </button>
        {agentMenuOpen && (
          <div className="agent-menu" onClick={() => setAgentMenuOpen(false)}>
            <div className="agent-menu__item" onClick={props.onNewAgent}>
              + New agent
            </div>
          </div>
        )}
      </div>

      {/* Project tree */}
      <div className="project-tree">
        <div className="project-tree__header">
          <span>{t.project}</span>
          <div className="project-tree__actions">
            <button title="Add project">＋</button>
            <button title="Refresh">⟳</button>
            <button title="List view">≡</button>
          </div>
        </div>

        {mockProjects.map((proj) => (
          <div key={proj.id} className="project">
            <div className="project__header">
              <span className="project__icon">▾</span>
              <span className="project__name">{proj.name}</span>
            </div>
            <div className="project__sessions">
              {proj.sessions
                .filter((s) => !props.search || s.title.toLowerCase().includes(props.search.toLowerCase()))
                .map((s) => (
                  <SessionItem
                    key={s.id}
                    session={s}
                    active={s.id === props.activeSessionId}
                    onClick={() => props.onSelectSession(s.id)}
                  />
                ))}
            </div>
          </div>
        ))}
      </div>
    </>
  )
}

function AgentTab(props: { agent: Agent; active: boolean; onClick: () => void }) {
  return (
    <button
      className={`agent-tab ${props.active ? 'active' : ''}`}
      onClick={props.onClick}
      title={props.agent.name}
    >
      <span className="agent-tab__name">{props.agent.name}</span>
      {props.agent.yolo && <span className="agent-tab__badge">YOLO</span>}
    </button>
  )
}

function SessionItem(props: { session: Session; active: boolean; onClick: () => void }) {
  return (
    <button
      className={`session-item ${props.active ? 'active' : ''}`}
      onClick={props.onClick}
    >
      <div className="session-item__title">{props.session.title}</div>
      <div className="session-item__meta">
        <span>{props.session.messageCount} 轮 · {props.session.toolCount}天前</span>
      </div>
    </button>
  )
}
