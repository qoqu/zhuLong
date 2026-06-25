import { useState, useMemo, useCallback, useEffect } from 'react'
import type { Language, GlobalInfo } from '../types'
import { resolveLanguage } from '../i18n'

interface DeletedSession {
  id: string
  originalId: string
  title: string
  globalName: string
  projectName: string
  globalId: string
  projectId: string
  dateKey: string
  timestamp: number
  messageCount: number
  toolCount: number
  preview: string
  deletedAt: string
}

interface RecycleBinPageProps {
  language: Language
  globals: GlobalInfo[]
  deletedSessions: DeletedSession[]
  onClose: () => void
  onRestore?: (trashId: string) => void
}

// Local storage key (for permanent delete within this component)
const TRASH_KEY = 'zhulong-trash-sessions'

function saveTrash(items: DeletedSession[]) {
  try { localStorage.setItem(TRASH_KEY, JSON.stringify(items)) } catch {}
}

export function RecycleBinPage(props: RecycleBinPageProps) {
  const isZh = resolveLanguage(props.language) === 'zh'
  const [searchQuery, setSearchQuery] = useState('')
  const [filterTab, setFilterTab] = useState<'all' | 'global' | 'project' | 'earlier'>('all')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  // Use real data from props (managed by App.tsx)
  const [items, setItems] = useState<DeletedSession[]>(() => {
    return props.deletedSessions || []
  })

  // Sync when props change (e.g. new session deleted from App)
  useEffect(() => {
    setItems(props.deletedSessions || [])
  }, [props.deletedSessions])

  // ── Update items and persist ──
  const updateItems = useCallback((updater: (prev: DeletedSession[]) => DeletedSession[]) => {
    setItems(prev => {
      const next = updater(prev)
      saveTrash(next)
      return next
    })
  }, [])

  // ── Filter ──
  const filteredItems = useMemo(() => {
    let result = [...items].sort((a, b) => b.timestamp - a.timestamp)

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(s =>
        s.title.toLowerCase().includes(q) ||
        s.globalName.toLowerCase().includes(q) ||
        s.projectName.toLowerCase().includes(q) ||
        s.preview.toLowerCase().includes(q)
      )
    }

    if (filterTab === 'earlier') {
      const today = new Date()
      result = result.filter(s => {
        const sDate = new Date(s.timestamp)
        return !(sDate.getFullYear() === today.getFullYear() &&
                 sDate.getMonth() === today.getMonth() &&
                 sDate.getDate() === today.getDate())
      })
    }

    return result
  }, [items, searchQuery, filterTab])

  // ── Group by date ──
  const groupedByDate = useMemo(() => {
    const groups: Map<string, DeletedSession[]> = new Map()
    for (const item of filteredItems) {
      const existing = groups.get(item.dateKey)
      if (existing) existing.push(item)
      else groups.set(item.dateKey, [item])
    }
    return groups
  }, [filteredItems])

  const selectedItem = selectedId ? items.find(i => i.id === selectedId) : null

  // ── Actions ──
  const handleRestore = (id: string) => {
    updateItems(prev => prev.filter(i => i.id !== id))
    setSelectedId(null)
    if (props.onRestore) props.onRestore(id)
  }

  const handleDeleteForever = (id: string) => {
    if (!isZh && !window.confirm('Permanently delete this session? This cannot be undone.')) return
    if (isZh && !window.confirm('确定要永久删除此会话？此操作不可撤销。')) return
    updateItems(prev => prev.filter(i => i.id !== id))
    setSelectedId(null)
  }

  const handleClearAll = () => {
    if (!isZh && !window.confirm(`Permanently delete all ${items.length} sessions? This cannot be undone.`)) return
    if (isZh && !window.confirm(`确定要清空回收站吗？将永久删除全部 ${items.length} 个会话，此操作不可撤销。`)) return
    updateItems(() => [])
    setSelectedId(null)
  }

  // Format date key nicely
  const formatDateKey = (key: string) => key

  return (
    <div className="history-overlay" onClick={props.onClose}>
      <div className="history-panel history-panel--trash" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="history-header">
          <h2 className="history-header__title">{isZh ? '回收站' : 'Recycle Bin'}</h2>
          <div className="history-header__actions">
            <button className="history-btn history-btn--ghost-sm" onClick={handleClearAll}
              disabled={items.length === 0}>
              {isZh ? '清空回收站' : 'Empty Trash'}
            </button>
            <button className="history-header__close" onClick={props.onClose}>✕</button>
          </div>
        </div>

        {/* Search */}
        <div className="history-search">
          <span className="history-search__icon">🔍</span>
          <input
            type="text"
            className="history-search__input"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder={isZh ? '搜索会话...' : 'Search sessions...'}
          />
        </div>

        {/* Filter Tabs */}
        <div className="history-filters">
          <HistoryFilterTab label={isZh ? '全部' : 'All'} count={items.length} active={filterTab === 'all'} onClick={() => setFilterTab('all')} />
          <HistoryFilterTab label={isZh ? '全局' : 'Global'} count={items.length} active={filterTab === 'global'} onClick={() => setFilterTab('global')} />
          <HistoryFilterTab label={isZh ? '全部' : 'All'} count={items.length} active={filterTab === 'project'} onClick={() => setFilterTab('project')} />
          <HistoryFilterTab label={isZh ? '更早' : 'Earlier'} count={Math.max(0, filteredItems.length)} active={filterTab === 'earlier'} onClick={() => setFilterTab('earlier')} />
        </div>

        {/* Main Content */}
        <div className="history-body">
          {/* Left - List */}
          <div className="history-list">
            {groupedByDate.size === 0 ? (
              <div className="history-empty">
                <span>🗑️</span>
                <p>{isZh ? '回收站为空' : 'Recycle bin is empty'}</p>
                <p style={{ fontSize: 12, color: 'var(--fg-faint)', marginTop: 4 }}>
                  {isZh ? '删除的对话会出现在这里' : 'Deleted conversations will appear here'}
                </p>
              </div>
            ) : Array.from(groupedByDate.entries()).map(([dateKey, sessions]) => (
              <div key={dateKey} className="history-date-group">
                <div className="history-date-group__header">
                  <span className="history-date-group__date">{formatDateKey(dateKey)}</span>
                  <span className="history-date-group__count">{sessions.length}</span>
                </div>
                {sessions.map(item => (
                  <div
                    key={item.id}
                    className={`history-session-item history-session-item--deleted ${selectedId === item.id ? 'history-session-item--active' : ''}`}
                    onClick={() => setSelectedId(item.id)}
                  >
                    <div className="history-session-item__title">{item.title}</div>
                    <div className="history-session-item__meta">
                      <span className={`history-session-item__status`}>
                        ⏳{isZh ? '已删除' : 'Deleted'}
                      </span>
                      <span className="history-session-item__ws">
                        <span className="history-session-item__ws-icon">📁</span>
                        {item.globalName || item.projectName}
                      </span>
                      <span className="history-session-item__stats">
                        {item.messageCount > 0 && <span>{item.messageCount} {isZh ? '轮' : 'turns'}</span>}
                        <span className="history-session-item__time">{item.deletedAt}</span>
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            ))}
          </div>

          {/* Right - Detail */}
          <div className="history-detail">
            {!selectedItem ? (
              <div className="history-detail__empty">
                <span>← {isZh ? '选择一个已删除的会话查看详情' : 'Select a deleted session to view details'}</span>
              </div>
            ) : (
              <>
                {/* Detail Header */}
                <div className="history-detail__header">
                  <div className="history-detail__title-row">
                    <h3 className="history-detail__title">{selectedItem.title}</h3>
                  </div>
                  <div className="history-detail__actions">
                    <button className="history-btn history-btn--primary" onClick={() => handleRestore(selectedItem.id)}>
                      {isZh ? '恢复' : 'Restore'}
                    </button>
                    <button className="history-btn history-btn--danger" onClick={() => handleDeleteForever(selectedItem.id)}>
                      {isZh ? '彻底删除' : 'Delete Forever'}
                    </button>
                  </div>
                </div>

                {/* Detail Body */}
                <div className="history-detail__body">
                  <div className="history-detail__preview">
                    <div className="history-detail__meta-line">
                      <span className={`history-detail__badge history-detail__badge--del`}>{isZh ? '已删除' : 'Deleted'}</span>
                      <span className={`history-detail__badge history-detail__badge--ws`}>{selectedItem.globalName}</span>
                      {selectedItem.messageCount > 0 && <span>{selectedItem.messageCount} {isZh ? '轮' : 'turns'}</span>}
                      <span className="history-detail__time-del">{selectedItem.deletedAt}</span>
                      <span className="history-detail__label-del">{isZh ? '已删除' : 'Deleted'}</span>
                    </div>

                    {selectedItem.preview && (
                      <p className="history-detail__preview-text trash-preview-text">{selectedItem.preview}</p>
                    )}

                    {/* Session content preview */}
                    <div className="trash-content">
                      <div className="trash-msg trash-msg--user">
                        <div className="trash-msg__content">{selectedItem.title}</div>
                      </div>

                      {selectedItem.preview && (
                        <>
                          <div className="trash-msg trash-msg--assistant">
                            <div className="trash-msg__content">{selectedItem.preview}</div>
                          </div>
                          {selectedItem.toolCount > 0 && (
                            <div className="trash-tool-call">
                              <div className="trash-tool-call__header">
                                <span className="trash-tool-call__icon">✓</span>
                                <code>tool_call executed</code>
                              </div>
                              <p className="trash-tool-result">
                                {isZh ? '任务已完成。' : 'Task completed.'}
                              </p>
                            </div>
                          )}
                        </>
                      )}

                      <div className="trash-msg trash-msg--assistant">
                        <div className="trash-msg__content">
                          <p><strong>{isZh ? '对话摘要' : 'Summary'}</strong></p>
                          <p style={{ color: 'var(--fg-soft)', fontSize: 13, lineHeight: 1.6 }}>
                            {selectedItem.preview || (isZh ? '此会话的内容预览暂不可用。' : 'Content preview not available for this session.')}
                          </p>
                          <table className="trash-table">
                            <thead>
                              <tr>
                                <th>{isZh ? '项目' : 'Project'}</th>
                                <th>{isZh ? '工作空间' : 'Workspace'}</th>
                                <th>{isZh ? '消息数' : 'Messages'}</th>
                                <th>{isZh ? '工具调用' : 'Tool Calls'}</th>
                                <th>{isZh ? '删除时间' : 'Deleted At'}</th>
                              </tr>
                            </thead>
                            <tbody>
                              <tr>
                                <td>{selectedItem.projectName}</td>
                                <td>{selectedItem.globalName}</td>
                                <td>{selectedItem.messageCount}</td>
                                <td>{selectedItem.toolCount}</td>
                                <td>{selectedItem.deletedAt}</td>
                              </tr>
                            </tbody>
                          </table>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Bottom toolbar */}
                  <div className="history-detail__toolbar">
                    <button className="history-toolbar-btn" onClick={() => handleRestore(selectedItem.id)} title={isZh ? '恢复此会话' : 'Restore this session'}>
                      ↩️
                    </button>
                    <button className="history-toolbar-btn" title={isZh ? '导出' : 'Export'}>
                      📋
                    </button>
                    <button className="history-toolbar-btn history-toolbar-btn--danger"
                      onClick={() => handleDeleteForever(selectedItem.id)} title={isZh ? '彻底删除' : 'Permanently delete'}>
                      🔥
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

// ─── Filter tab button ───
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
