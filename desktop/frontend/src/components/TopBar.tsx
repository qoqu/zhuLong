import type { Language } from '../types'
import { resolveLanguage } from '../i18n'

interface SessionTab {
  id: string
  title: string
}

interface TopBarProps {
  language: Language
  sessions: SessionTab[]
  activeSessionId: string
  onSwitchSession: (id: string) => void
  onCloseSession?: (id: string) => void
  onToggleSidebar: () => void
  onToggleRightPanel: () => void
}

export function TopBar(props: TopBarProps) {
  const isZh = resolveLanguage(props.language) === 'zh'
  return (
    <div className="topbar">
      <button className="topbar__toggle" onClick={props.onToggleSidebar} title={isZh ? '侧边栏' : 'Sidebar'}>
        ☰
      </button>
      <div className="topbar__tabs">
        {props.sessions.map((s) => (
          <div
            key={s.id}
            className={`topbar__tab ${s.id === props.activeSessionId ? 'active' : ''}`}
            onClick={() => props.onSwitchSession(s.id)}
          >
            <span className="topbar__tab-title">{s.title}</span>
            {props.onCloseSession && (
              <button
                className="topbar__tab-close"
                onClick={(e) => { e.stopPropagation(); props.onCloseSession?.(s.id) }}
                title={isZh ? '关闭' : 'Close'}
              >
                ×
              </button>
            )}
          </div>
        ))}
      </div>
      <button className="topbar__toggle" onClick={props.onToggleRightPanel} title={isZh ? '右侧栏' : 'Right Panel'}>
        ☰
      </button>
    </div>
  )
}
