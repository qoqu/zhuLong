import type { Language } from '../types'

interface TopBarProps {
  language: Language
  sessionTitle: string
  sessionScope: string
  onRename: () => void
  onExport: () => void
}

export function TopBar(props: TopBarProps) {
  return (
    <div className="topbar">
      <div className="topbar__left">
        <div className="topbar__title">{props.sessionTitle}</div>
        <button className="topbar__icon-btn" title="Rename" onClick={props.onRename}>
          ✎
        </button>
      </div>
      <div className="topbar__scope">{props.sessionScope}</div>
      <div className="topbar__spacer" />
      <div className="topbar__actions">
        <button className="topbar__icon-btn" title="Copy" onClick={() => navigator.clipboard?.writeText(props.sessionTitle)}>
          ⎘
        </button>
        <button className="topbar__icon-btn" title="Download" onClick={props.onExport}>
          ⤓
        </button>
        <button className="topbar__icon-btn" title="Branch">⑂</button>
        <button className="topbar__icon-btn" title="Settings">⚙</button>
      </div>
    </div>
  )
}
