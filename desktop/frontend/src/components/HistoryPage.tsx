import { useState, useMemo, useCallback } from 'react'
import type { Language, SessionInfo, GlobalInfo, Message } from '../types'
import { resolveLanguage } from '../i18n'

interface HistoryPageProps {
  language: Language
  globals: GlobalInfo[]
  activeSessionId: string
  onClose: () => void
  onSelectSession: (id: string) => void
  onOpenAll: (id: string) => void
  onMoveToTrash: (id: string) => void
  onRename: (id: string) => void
}

// ─── 增强的会话条目（带日期和完整信息）───
interface SessionEntry extends SessionInfo {
  globalName: string
  globalId: string
  projectName: string
  projectId: string
  dateKey: string       // "2026/6/16"
  dateLabel: string     // "2026/6/16"
  timestamp: number      // for sorting
  messages?: Message[]
}

// ─── 历史页面主组件 ───
export function HistoryPage(props: HistoryPageProps) {
  const isZh = resolveLanguage(props.language) === 'zh'
  const [searchQuery, setSearchQuery] = useState('')
  const [filterTab, setFilterTab] = useState<'all' | 'global' | 'project' | 'current' | 'today' | 'earlier'>('all')
  const [selectedId, setSelectedId] = useState<string | null>(props.activeSessionId || null)

  // ── 收集所有会话并分组 ──
  const allSessions: SessionEntry[] = useMemo(() => {
    const entries: SessionEntry[] = []
    for (const g of props.globals) {
      for (const p of g.projects) {
        for (const s of p.sessions) {
          const dateObj = parseDate(s.updatedAt)
          entries.push({
            ...s,
            globalName: g.name,
            globalId: g.id,
            projectName: p.name,
            projectId: p.id,
            dateKey: formatDateKey(dateObj),
            dateLabel: formatDateLabel(dateObj),
            timestamp: dateObj.getTime(),
          })
        }
      }
    }
    return entries.sort((a, b) => b.timestamp - a.timestamp)
  }, [props.globals])

  // ── 过滤逻辑 ──
  const filteredSessions = useMemo(() => {
    let result = allSessions

    // 搜索过滤
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(s =>
        s.title.toLowerCase().includes(q) ||
        s.globalName.toLowerCase().includes(q) ||
        s.projectName.toLowerCase().includes(q) ||
        (s.preview && s.preview.toLowerCase().includes(q))
      )
    }

    // 标签过滤
    switch (filterTab) {
      case 'current':
        result = result.filter(s => s.id === props.activeSessionId)
        break
      case 'today': {
        const today = new Date()
        const todayKey = `${today.getFullYear()}/${today.getMonth() + 1}/${today.getDate()}`
        result = result.filter(s => s.dateKey === todayKey)
        break
      }
      case 'global':
        // 按工作空间分组显示（这里简单处理为全部）
        break
      case 'project':
        break
      default:
        break
    }

    return result
  }, [allSessions, searchQuery, filterTab, props.activeSessionId])

  // ── 按日期分组 ──
  const groupedByDate = useMemo(() => {
    const groups: Map<string, SessionEntry[]> = new Map()
    for (const s of filteredSessions) {
      const existing = groups.get(s.dateKey)
      if (existing) existing.push(s)
      else groups.set(s.dateKey, [s])
    }
    return groups
  }, [filteredSessions])

  // 统计数字
  const totalCount = allSessions.length
  const todayCount = (() => {
    const today = new Date()
    const todayKey = `${today.getFullYear()}/${today.getMonth() + 1}/${today.getDate()}`
    return allSessions.filter(s => s.dateKey === todayKey).length
  })()
  const currentCount = selectedId ? 1 : 0

  // 选中的会话
  const selectedSession = selectedId ? allSessions.find(s => s.id === selectedId) : null

  const handleSelect = useCallback((id: string) => {
    setSelectedId(id)
  }, [])

  const handleOpenAll = () => {
    if (selectedId) {
      props.onSelectSession(selectedId)
      props.onClose()
    }
  }

  return (
    <div className="history-overlay" onClick={props.onClose}>
      <div className="history-panel" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="history-header">
          <h2 className="history-header__title">{isZnZz(isZh, '历史', 'History')}</h2>
          <button className="history-header__close" onClick={props.onClose}>✕</button>
        </div>

        {/* Search */}
        <div className="history-search">
          <span className="history-search__icon">🔍</span>
          <input
            type="text"
            className="history-search__input"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder={isZnZz(isZh, '搜索会话...', 'Search sessions...')}
          />
        </div>

        {/* Filter Tabs */}
        <div className="history-filters">
          <HistoryFilterTab label={isZnZz(isZh, '全部', 'All')} count={totalCount} active={filterTab === 'all'} onClick={() => setFilterTab('all')} />
          <HistoryFilterTab label={isZnZz(isZh, '全局', 'Global')} count={totalCount} active={filterTab === 'global'} onClick={() => setFilterTab('global')} />
          <HistoryFilterTab label={isZnZz(isZh, '全部', 'All')} count={totalCount} active={filterTab === 'project'} onClick={() => setFilterTab('project')} />
          <HistoryFilterTab label={isZnZz(isZh, '当前', 'Current')} count={currentCount} active={filterTab === 'current'} onClick={() => setFilterTab('current')} />
          <HistoryFilterTab label={isZnZz(isZh, '全部', 'All')} count={todayCount} active={filterTab === 'today'} onClick={() => setFilterTab('today')} />
          <HistoryFilterTab label={isZnZz(isZh, '更早', 'Earlier')} count={Math.max(0, totalCount - todayCount)} active={filterTab === 'earlier'} onClick={() => setFilterTab('earlier')} />
        </div>

        {/* Main Content */}
        <div className="history-body">
          {/* Left - Session List */}
          <div className="history-list">
            {groupedByDate.size === 0 ? (
              <div className="history-empty">
                <span>{isZnZz(isZh, '没有匹配的会话', 'No matching sessions')}</span>
              </div>
            ) : Array.from(groupedByDate.entries()).map(([dateKey, sessions]) => (
              <div key={dateKey} className="history-date-group">
                <div className="history-date-group__header">
                  <span className="history-date-group__date">{dateKey}</span>
                  <span className="history-date-group__count">{sessions.length}</span>
                </div>
                {sessions.map(session => (
                  <div
                    key={session.id}
                    className={`history-session-item ${selectedId === session.id ? 'history-session-item--active' : ''}`}
                    onClick={() => handleSelect(session.id)}
                  >
                    <div className="history-session-item__title">{session.title}</div>
                    <div className="history-session-item__meta">
                      <span className="history-session-item__ws">
                        <span className="history-session-item__ws-icon">📁</span>
                        {session.globalName}
                      </span>
                      <span className="history-session-item__stats">
                        {session.messageCount > 0 && <span>{session.messageCount} {isZnZz(isZh, '轮', 'turns')}</span>}
                        <span className="history-session-item__time">{session.updatedAt}</span>
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            ))}
          </div>

          {/* Right - Detail */}
          <div className="history-detail">
            {!selectedSession ? (
              <div className="history-detail__empty">
                <span>← {isZnZz(isZh, '选择一个会话查看详情', 'Select a session to view details')}</span>
              </div>
            ) : (
              <>
                {/* Detail Header */}
                <div className="history-detail__header">
                  <div className="history-detail__title-row">
                    <h3 className="history-detail__title">{selectedSession.title}</h3>
                  </div>
                  <div className="history-detail__actions">
                    <button className="history-btn history-btn--primary" onClick={handleOpenAll}>
                      {isZnZz(isZh, '打开全部', 'Open All')}
                    </button>
                    <button className="history-btn history-btn--ghost" onClick={() => {
                      props.onRename(selectedSession.id)
                      props.onClose()
                    }}>
                      {isZnZz(isZh, '重命名', 'Rename')}
                    </button>
                    <button className="history-btn history-btn--danger" onClick={() => {
                      props.onMoveToTrash(selectedSession.id)
                      setSelectedId(null)
                    }}>
                      {isZnZz(isZh, '移到回收站', 'Move to Trash')}
                    </button>
                  </div>
                </div>

                {/* Detail Body - Preview */}
                <div className="history-detail__body">
                  <div className="history-detail__preview">
                    <div className="history-detail__meta-line">
                      <span className={`history-detail__badge history-detail__badge--ws`}>{selectedSession.globalName}</span>
                      <span className="history-detail__badge history-detail__badge--proj">{selectedSession.projectName}</span>
                      {selectedSession.messageCount > 0 && <span>{selectedSession.messageCount} {isZnZz(isZh, '轮', 'turns')}</span>}
                      <span>{selectedSession.updatedAt}</span>
                    </div>
                    {selectedSession.preview && (
                      <p className="history-detail__preview-text">{selectedSession.preview}</p>
                    )}
                    <div className="history-detail__placeholder">
                      <div className="history-detail__placeholder-icon">💬</div>
                      <p>{isZnZz(isZh, '点击「打开全部」在聊天窗口中查看完整对话内容。', 'Click "Open All" to view the full conversation in the chat window.')}</p>
                    </div>
                  </div>

                  {/* Bottom toolbar */}
                  <div className="history-detail__toolbar">
                    <button className="history-toolbar-btn" title={isZnZz(isZh, '分享全部', 'Share all')}>
                      📤
                    </button>
                    <button className="history-toolbar-btn" title={isZnZz(isZh, '导出会话', 'Export session')}>
                      📋
                    </button>
                    <button className="history-toolbar-btn" title={isZnZz(isZh, '复制链接', 'Copy link')}>
                      🔗
                    </button>
                    <button className="history-toolbar-btn" title={isZnZz(isZh, '删除', 'Delete')}>
                      🗑️
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ─── 过滤标签按钮 ───
function HistoryFilterTab({ label, count, active, onClick }: {
  label: string; count: number; active: boolean; onClick: () => void
}) {
  return (
    <button
      className={`history-filter-tab ${active ? 'active' : ''}`}
      onClick={onClick}
    >
      {label}
      <span className="history-filter-tab__count">{count}</span>
    </button>
  )
}

// ════════════════════════════════
// 工具函数
// ════════════════════════════════

function isZnZz(isZh: boolean, zn: string, en: string): string {
  return isZh ? zn : en
}

function parseDate(dateStr: string): Date {
  // 尝试解析各种日期格式
  if (!dateStr) return new Date()

  // 刚刚 / 几分钟前 等
  if (dateStr.includes('刚刚') || dateStr === 'now') return new Date()

  const d = new Date(dateStr)
  if (!isNaN(d.getTime())) return d

  // 中文相对时间 fallback
  return new Date()
}

function formatDateKey(d: Date): string {
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

function formatDateLabel(d: Date): string {
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}
