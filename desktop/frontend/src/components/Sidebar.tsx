import { useState, useRef, useEffect } from 'react'
import type { Language, AgentInfo, GlobalInfo, SidebarView, ExecutionMode } from '../types'
import { useT, resolveLanguage } from '../i18n'

// ════════════════════════════════════════
// Types
// ════════════════════════════════════════

type ContextMenuTarget =
  | { type: 'session'; id: string; title: string }
  | { type: 'workspace'; id: string; name: string; color?: string }
  | { type: 'project'; id: string; name: string; globalId: string; color?: string }
  | null

interface SidebarProps {
  language: Language
  onLanguageChange: (l: Language) => void
  darkMode: boolean
  onToggleDarkMode: () => void
  collapsed: boolean
  onToggleCollapsed: () => void
  view: SidebarView
  onChangeView: (v: SidebarView) => void
  onOpenSettings: () => void
  desktopStyle?: 'classic' | 'workspace'
  executionMode?: ExecutionMode
  agents: AgentInfo[]
  globals: GlobalInfo[]
  activeGlobalId: string
  activeProjectId: string
  activeAgentId: string
  onSelectAgent: (id: string) => void
  onSelectGlobal: (id: string) => void
  onSelectProject: (id: string) => void
  activeSessionId: string
  onSelectSession: (id: string) => void
  onNewSession: (projectId?: string) => void
  onDeleteSession: (id: string) => void
  onRenameSession: (id: string) => void
  onCreateGlobal: () => void
  onCreateProject: (globalId: string) => void
  onOpenHistory?: () => void
  onOpenTrash?: () => void
}

// ════════════════════════════════════════
// Color options for workspace/project
// ════════════════════════════════════════

const COLOR_OPTIONS = [
  { key: 'default', labelZh: '默认', labelEn: 'Default', dot: '#e87a3f' },
  { key: 'red', labelZh: '红色', labelEn: 'Red', dot: '#ef4444' },
  { key: 'orange', labelZh: '橙色', labelEn: 'Orange', dot: '#f97316' },
  { key: 'amber', labelZh: '琥珀', labelEn: 'Amber', dot: '#f59e0b' },
  { key: 'green', labelZh: '绿色', labelEn: 'Green', dot: '#22c55e' },
  { key: 'cyan', labelZh: '青色', labelEn: 'Cyan', dot: '#06b6d4' },
  { key: 'blue', labelZh: '蓝色', labelEn: 'Blue', dot: '#3b82f6' },
  { key: 'purple', labelZh: '紫色', labelEn: 'Purple', dot: '#a855f7' },
  { key: 'pink', labelZh: '粉色', labelEn: 'Pink', dot: '#ec4899' },
]

// ════════════════════════════════════════
// Context Menu Component
// ════════════════════════════════════════

function ContextMenu({
  x,
  y,
  target,
  isZh,
  onClose,
  onRename,
  onMoveToTrash,
  onChangeColor,
  onShowInExplorer,
  onCopyPath,
}: {
  x: number
  y: number
  target: ContextMenuTarget
  isZh: boolean
  onClose: () => void
  onRename: () => void
  onMoveToTrash: () => void
  onChangeColor: (colorKey: string) => void
  onShowInExplorer: () => void
  onCopyPath: () => void
}) {
  const menuRef = useRef<HTMLDivElement>(null)

  // Close on click outside
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  // Adjust position to stay within viewport
  const adjustedX = Math.min(x, window.innerWidth - 240)
  const adjustedY = Math.min(y, window.innerHeight - (target?.type === 'session' ? 120 : 450))

  if (!target) return null

  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{ left: adjustedX, top: adjustedY }}
    >
      {/* ── Session Menu ── */}
      {target.type === 'session' && (
        <>
          <button className="context-menu__item" onClick={() => { onRename(); onClose() }}>
            <span className="context-menu__icon">✎</span>
            <span>{isZh ? '重命名会话' : 'Rename'}</span>
          </button>
          <button className="context-menu__item context-menu__item--danger" onClick={() => { onMoveToTrash(); onClose() }}>
            <span className="context-menu__icon">🗑</span>
            <span>{isZh ? '移到回收站' : 'Move to Trash'}</span>
          </button>
        </>
      )}

      {/* ── Workspace / Project Menu ── */}
      {(target.type === 'workspace' || target.type === 'project') && (
        <>
          {/* Rename */}
          <button className="context-menu__item" onClick={() => { onRename(); onClose() }}>
            <span className="context-menu__icon">✎</span>
            <span>{isZh ? '修改显示名称' : 'Rename'}</span>
          </button>

          {/* Separator */}
          <div className="context-menu__separator" />

          {/* Color picker */}
          <div className="context-menu__color-section">
            {COLOR_OPTIONS.map(opt => (
              <button
                key={opt.key}
                className={`context-menu__color-item ${target.color === opt.key || (!target.color && opt.key === 'default') ? 'active' : ''}`}
                onClick={() => { onChangeColor(opt.key); onClose() }}
              >
                <span className="context-menu__color-dot" style={{ background: opt.dot }} />
                <span className="context-menu__color-label">{isZh ? opt.labelZh : opt.labelEn}</span>
                {(target.color === opt.key || (!target.color && opt.key === 'default')) && (
                  <span className="context-menu__check">✓</span>
                )}
              </button>
            ))}
          </div>

          {/* Separator */}
          <div className="context-menu__separator" />

          {/* Show in Explorer */}
          <button className="context-menu__item" onClick={() => { onShowInExplorer(); onClose() }}>
            <span className="context-menu__icon">📁</span>
            <span>{isZh ? '在文件资源管理器中显示' : 'Show in Explorer'}</span>
          </button>

          {/* Copy path */}
          <button className="context-menu__item" onClick={() => { onCopyPath(); onClose() }}>
            <span className="context-menu__icon">📋</span>
            <span>{isZh ? '复制路径' : 'Copy Path'}</span>
          </button>
        </>
      )}
    </div>
  )
}

// ════════════════════════════════════════
// Sidebar Component
// ════════════════════════════════════════

export function Sidebar(props: SidebarProps) {
  const t = useT(props.language)
  const isZh = resolveLanguage(props.language) === 'zh'

  // Search input state
  const [searchQuery, setSearchQuery] = useState('')
  // Global expand/collapse state
  const [globalExpanded, setGlobalExpanded] = useState(true)

  // ── Context menu state ──
  const [ctxMenu, setCtxMenu] = useState<{
    x: number
    y: number
    target: ContextMenuTarget
  } | null>(null)

  // Color state for workspaces/projects (stored by id, persisted to localStorage)
  const [itemColors, setItemColors] = useState<Record<string, string>>(() => {
    try { const v = localStorage.getItem('zhulong-item-colors'); return v ? JSON.parse(v) : {} } catch { return {} }
  })

  // Persist colors to localStorage
  useEffect(() => {
    try { localStorage.setItem('zhulong-item-colors', JSON.stringify(itemColors)) } catch {}
  }, [itemColors])

  // Close context menu on Escape or scroll
  useEffect(() => {
    if (!ctxMenu) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setCtxMenu(null)
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [ctxMenu])

  // ── Context menu handlers ──
  const handleContextMenu = (e: React.MouseEvent, target: ContextMenuTarget) => {
    e.preventDefault()
    e.stopPropagation()
    setCtxMenu({ x: e.clientX, y: e.clientY, target })
  }

  const closeCtxMenu = () => setCtxMenu(null)

  const handleSessionAction = (action: string, sessionId: string) => {
    switch (action) {
      case 'rename':
        props.onRenameSession(sessionId)
        break
      case 'trash':
        props.onDeleteSession(sessionId)
        break
    }
  }

  const handleWorkspaceAction = async (action: string, workspaceId: string) => {
    switch (action) {
      case 'newSession':
        // Find first project under this workspace and create session there
        const ws = props.globals.find(g => g.id === workspaceId)
        if (ws && ws.projects.length > 0) {
          props.onNewSession(ws.projects[0].id)
        }
        break
      case 'showInExplorer':
        try {
          const backend = (window as any).go?.main?.App
          if (backend) {
            const path = await backend.GetGlobalPath(workspaceId)
            if (path) await backend.OpenInExplorer(path)
          }
        } catch (e) { console.error('Failed to open in explorer:', e) }
        break
      case 'copyPath':
        try {
          const backend = (window as any).go?.main?.App
          if (backend) {
            const path = await backend.GetGlobalPath(workspaceId)
            if (path) navigator.clipboard.writeText(path)
          } else {
            navigator.clipboard.writeText(`/workspaces/${workspaceId}`)
          }
        } catch { navigator.clipboard.writeText(`/workspaces/${workspaceId}`) }
        break
    }
  }

  const handleProjectAction = async (action: string, projectId: string, globalId: string) => {
    switch (action) {
      case 'newSession':
        props.onNewSession(projectId)
        break
      case 'showInExplorer':
        try {
          const backend = (window as any).go?.main?.App
          if (backend) {
            const path = await backend.GetProjectPath(globalId, projectId)
            if (path) await backend.OpenInExplorer(path)
          }
        } catch (e) { console.error('Failed to open in explorer:', e) }
        break
      case 'copyPath':
        try {
          const backend = (window as any).go?.main?.App
          if (backend) {
            const path = await backend.GetProjectPath(globalId, projectId)
            if (path) navigator.clipboard.writeText(path)
          } else {
            navigator.clipboard.writeText(`/workspaces/${globalId}/${projectId}`)
          }
        } catch { navigator.clipboard.writeText(`/workspaces/${globalId}/${projectId}`) }
        break
    }
  }

  const handleChangeColor = (itemId: string, colorKey: string) => {
    setItemColors(prev => ({
      ...prev,
      [itemId]: colorKey === 'default' ? '' : colorKey,
    }))
  }

  if (props.collapsed) {
    return (
      <aside className="sidebar sidebar--collapsed">
        <button className="sidebar__expand-btn" onClick={props.onToggleCollapsed} title={isZh ? '展开' : 'Expand'}>
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
          <button className="sidebar__icon-btn" title={t.newSession} onClick={() => props.onNewSession()}>
            ✚
          </button>
        </div>
      </aside>
    )
  }

  return (
    <aside className="sidebar">

      {/* ══ Brand Name ══ */}
      <div className="sidebar__brand" onClick={props.onToggleCollapsed}>
        <span className="sidebar__brand-icon">Z</span>
        <span className="sidebar__brand-name">烛龙</span>
      </div>

      {/* ══ Orange New Session Button (full width) ══ */}
      <button className="sidebar__new-session-btn" onClick={() => props.onNewSession()}>
        <span className="sidebar__new-session-btn__icon">✎</span>
        <span>{t.newSession}</span>
      </button>

      {/* ══ Search Bar ══ */}
      <div className="sidebar__search">
        <span className="sidebar__search__icon">🔍</span>
        <input
          className="sidebar__search__input"
          placeholder={isZh ? '搜索项目或会话' : 'Search projects or sessions'}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {/* ══ Body ══ */}
      <div className="sidebar__body">

        {/* Projects View */}
        {props.view === 'projects' && (
          <>
            {/* Section header: 项目 + action icons */}
            <div className="sidebar__section-header">
              <span className="sidebar__section-title">{isZh ? '项目' : 'Projects'}</span>
              <button
                className="sidebar__section-action-btn sidebar__section-add-btn"
                onClick={props.onCreateGlobal}
                title={isZh ? '新建工作空间' : 'New Workspace'}
              >+</button>
            </div>

            {/* Tree: Workspace → Project → Session */}
            <div className="tree-view">
              {props.globals.map(g => {
                const wsColor = itemColors[g.id]
                const colorDotStyle = wsColor
                  ? COLOR_OPTIONS.find(c => c.key === wsColor)?.dot
                  : undefined

                return (
                  <div key={g.id} className="tree-workspace">
                    {/* Global row: ▾/▸ ● Global + */}
                    <div
                      className="tree-global-row"
                      onClick={() => setGlobalExpanded(!globalExpanded)}
                      onContextMenu={(e) => handleContextMenu(e, {
                        type: 'workspace',
                        id: g.id,
                        name: g.name,
                        color: itemColors[g.id],
                      })}
                    >
                      <span className="tree-global-row__chevron">{globalExpanded ? '▾' : '▸'}</span>
                      <span
                        className={`tree-global-row__dot ${g.id === props.activeGlobalId ? 'active' : ''}`}
                        style={colorDotStyle ? { backgroundColor: colorDotStyle, borderColor: colorDotStyle } : undefined}
                      />
                      <span className="tree-global-row__label">{g.name}</span>
                      <button
                        className="tree-global-row__add"
                        onClick={(e) => { e.stopPropagation(); props.onCreateProject(g.id) }}
                        title={isZh ? '添加工作区' : 'Add Project'}
                      >+</button>
                    </div>

                    {/* Expanded content: Projects → Sessions */}
                    {globalExpanded && (
                      <div className="tree-global-children">
                        {g.projects.map(p => {
                          const projColor = itemColors[p.id]
                          const projDotStyle = projColor
                            ? COLOR_OPTIONS.find(c => c.key === projColor)?.dot
                            : undefined

                          return (
                            <div key={p.id} className="tree-project-group">
                              {/* Project row (name + new session) */}
                              <div
                                className="tree-project-row"
                                onContextMenu={(e) => handleContextMenu(e, {
                                  type: 'project',
                                  id: p.id,
                                  name: p.name,
                                  globalId: g.id,
                                  color: itemColors[p.id],
                                })}
                              >
                                <span
                                  className={`tree-project-row__name ${p.id === props.activeProjectId ? 'active' : ''}`}
                                  onClick={() => props.onSelectProject(p.id)}
                                  style={projDotStyle ? { color: projDotStyle, borderLeftColor: projDotStyle } : undefined}
                                >{p.name}</span>
                                <button
                                  className="tree-project-row__add"
                                  onClick={(e) => { e.stopPropagation(); props.onNewSession(p.id) }}
                                  title={isZh ? '新建对话' : 'New Session'}
                                >+</button>
                              </div>

                              {/* Sessions under this project */}
                              {(p.sessions || []).map(s => {
                                if (searchQuery && !s.title.toLowerCase().includes(searchQuery.toLowerCase())) return null
                                const isActive = s.id === props.activeSessionId
                                return (
                                  <div
                                    key={s.id}
                                    className={`tree-session ${isActive ? 'active' : ''}`}
                                    onClick={() => props.onSelectSession(s.id)}
                                    onContextMenu={(e) => handleContextMenu(e, {
                                      type: 'session',
                                      id: s.id,
                                      title: s.title,
                                    })}
                                  >
                                    <span className="tree-session__title">{s.title}</span>
                                    <span className="tree-session__meta">
                                      {s.messageCount} {isZh ? '轮' : 'turns'} · {s.updatedAt}
                                    </span>
                                  </div>
                                )
                              })}
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </div>
                )
              })}

              {/* No data hint */}
              {(!props.globals || props.globals.length === 0) && (
                <div className="sidebar__empty-hint">{isZh ? '暂无项目' : 'No projects'}</div>
              )}
            </div>
          </>
        )}

        {/* Agents View */}
        {props.view === 'agents' && (
          <div className="agents-view">
            <h3 className="agents-view__title">{t.agents}</h3>
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
              </div>
            ))}
          </div>
        )}

        {/* History View */}
        {props.view === 'history' && (() => {
          const allSessions = props.globals.flatMap(g =>
            g.projects.flatMap(p => (p.sessions || []).map(s => ({ ...s, projectName: p.name })))
          )
          return (
            <div className="history-view">
              <h3 className="history-view__title">{t.history}</h3>
              {allSessions.slice(0, 20).map(s => (
                <div
                  key={s.id}
                  className={`history-view__item ${props.activeSessionId === s.id ? 'active' : ''}`}
                  onClick={() => props.onSelectSession(s.id)}
                >
                  <div className="history-view__title-text">{s.title}</div>
                  <div className="history-view__meta">
                    {s.messageCount} {isZh ? '轮' : 'msgs'} · {s.updatedAt}
                  </div>
                </div>
              ))}
            </div>
          )
        })()}

        {/* Trash View */}
        {props.view === 'trash' && (
          <div className="trash-view">
            <h3 className="trash-view__title">{t.trash}</h3>
            <p>{isZh ? '回收站为空' : 'Trash is empty'}</p>
          </div>
        )}
      </div>

      {/* Footer nav */}
      <div className="sidebar__footer">
        <NavItem icon="🕐" label={t.history} active={false} onClick={() => props.onOpenHistory ? props.onOpenHistory() : props.onChangeView('history')} />
        <NavItem icon="🗑" label={t.trash} active={false} onClick={() => props.onOpenTrash ? props.onOpenTrash() : props.onChangeView('trash')} />
        <NavItem icon="⚙" label={t.settings} active={false} onClick={props.onOpenSettings} />
      </div>

      {/* ══ Context Menu Overlay ══ */}
      {ctxMenu && (
        <ContextMenu
          x={ctxMenu.x}
          y={ctxMenu.y}
          target={ctxMenu.target}
          isZh={isZh}
          onClose={closeCtxMenu}
          onRename={() => {
            const t = ctxMenu.target!
            if (t.type === 'session') handleSessionAction('rename', t.id)
            else if (t.type === 'workspace') handleWorkspaceAction('rename', t.id)
            else if (t.type === 'project') handleProjectAction('rename', t.id, t.globalId)
          }}
          onMoveToTrash={() => {
            const t = ctxMenu.target!
            if (t.type === 'session') handleSessionAction('trash', t.id)
          }}
          onChangeColor={(colorKey) => {
            const t = ctxMenu.target!
            handleChangeColor(t.id, colorKey)
          }}
          onShowInExplorer={() => {
            const t = ctxMenu.target!
            if (t.type === 'workspace') handleWorkspaceAction('showInExplorer', t.id)
            else if (t.type === 'project') handleProjectAction('showInExplorer', t.id, t.globalId)
          }}
          onCopyPath={() => {
            const t = ctxMenu.target!
            if (t.type === 'workspace') handleWorkspaceAction('copyPath', t.id)
            else if (t.type === 'project') handleProjectAction('copyPath', t.id, t.globalId)
          }}
        />
      )}
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
