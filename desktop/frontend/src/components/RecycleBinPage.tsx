import { useState, useMemo, useCallback, useEffect } from 'react'
import type { Language, GlobalInfo } from '../types'
import { resolveLanguage } from '../i18n'

export interface DeletedItem {
  id: string
  type: 'global' | 'project' | 'session'
  name: string
  globalId?: string
  globalName?: string
  projectId?: string
  projectName?: string
  messageCount?: number
  toolCount?: number
  preview?: string
  dateKey: string
  timestamp: number
  deletedAt: string
}

interface RecycleBinPageProps {
  language: Language
  globals: GlobalInfo[]
  deletedSessions: DeletedItem[]
  onClose: () => void
  onRestore?: (item: DeletedItem) => void
}

// LocalStorage key for trash persistence
const TRASH_KEY = 'zhulong-trash-sessions'

function saveTrash(items: DeletedItem[]) {
  try { localStorage.setItem(TRASH_KEY, JSON.stringify(items)) } catch {}
}

export function RecycleBinPage(props: RecycleBinPageProps) {
  const isZh = resolveLanguage(props.language) === 'zh'
  const [searchQuery, setSearchQuery] = useState('')
  const [filterTab, setFilterTab] = useState<'all' | 'global' | 'project' | 'session'>('all')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [items, setItems] = useState<DeletedItem[]>(props.deletedSessions || [])

  useEffect(() => {
    setItems(props.deletedSessions || [])
  }, [props.deletedSessions])

  const filteredItems = useMemo(() => {
    let result = [...items].sort((a, b) => b.timestamp - a.timestamp)
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(s =>
        s.name.toLowerCase().includes(q) ||
        (s.globalName || '').toLowerCase().includes(q) ||
        (s.projectName || '').toLowerCase().includes(q)
      )
    }
    if (filterTab !== 'all') {
      result = result.filter(s => s.type === filterTab)
    }
    return result
  }, [items, searchQuery, filterTab])

  const groupedByDate = useMemo(() => {
    const groups: Map<string, DeletedItem[]> = new Map()
    for (const item of filteredItems) {
      const existing = groups.get(item.dateKey)
      if (existing) existing.push(item)
      else groups.set(item.dateKey, [item])
    }
    return groups
  }, [filteredItems])

  const selectedItem = selectedId ? items.find(i => i.id === selectedId) : null

  const handleRestore = useCallback(async (item: DeletedItem) => {
    // 调用后端恢复
    try {
      const backend = (window as any).go?.main?.App
      if (backend?.RestoreFromRecycleBin) {
        await backend.RestoreFromRecycleBin(item.id, item.type)
      }
    } catch (e) { console.error('Restore failed:', e) }

    // 从本地列表移除
    setItems(prev => prev.filter(i => i.id !== item.id))
    setSelectedId(null)
    if (props.onRestore) props.onRestore(item)
  }, [props.onRestore])

  const handleDeleteForever = useCallback(async (item: DeletedItem) => {
    const msg = isZh ? `确定要永久删除「${item.name}」？此操作不可撤销。` : `Permanently delete "${item.name}"? This cannot be undone.`
    if (!window.confirm(msg)) return

    // 如果是对话，从 localStorage 清理
    // 工作空间/项目的文件夹已在创建时移到回收站，这里只需清理记录
    setItems(prev => prev.filter(i => i.id !== item.id))
    setSelectedId(null)
  }, [isZh])

  const handleClearAll = useCallback(async () => {
    const msg = isZh ? `确定要清空回收站吗？将永久删除全部 ${items.length} 个项目，此操作不可撤销。` : `Permanently delete all ${items.length} items? This cannot be undone.`
    if (!window.confirm(msg)) return

    // 调用后端清空
    try {
      const backend = (window as any).go?.main?.App
      if (backend?.EmptyRecycleBin) {
        await backend.EmptyRecycleBin()
      }
    } catch (e) { console.error('Empty recycle bin failed:', e) }

    // 清空 React 状态和 localStorage
    setItems([])
    setSelectedId(null)
    saveTrash([])
  }, [items.length, isZh])

  const typeIcon = (type: string) => {
    switch (type) {
      case 'global': return '🏢'
      case 'project': return '📁'
      case 'session': return '💬'
      default: return '📄'
    }
  }

  const typeName = (type: string) => {
    switch (type) {
      case 'global': return isZh ? '工作空间' : 'Workspace'
      case 'project': return isZh ? '工作区' : 'Project'
      case 'session': return isZh ? '对话' : 'Session'
      default: return type
    }
  }

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
            placeholder={isZh ? '搜索...' : 'Search...'}
          />
        </div>

        {/* Filter Tabs */}
        <div className="history-filters">
          <HistoryFilterTab label={isZh ? '全部' : 'All'} count={items.length} active={filterTab === 'all'} onClick={() => setFilterTab('all')} />
          <HistoryFilterTab label={isZh ? '工作空间' : 'Workspace'} count={items.filter(i => i.type === 'global').length} active={filterTab === 'global'} onClick={() => setFilterTab('global')} />
          <HistoryFilterTab label={isZh ? '工作区' : 'Project'} count={items.filter(i => i.type === 'project').length} active={filterTab === 'project'} onClick={() => setFilterTab('project')} />
          <HistoryFilterTab label={isZh ? '对话' : 'Session'} count={items.filter(i => i.type === 'session').length} active={filterTab === 'session'} onClick={() => setFilterTab('session')} />
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
                  {isZh ? '删除的项目会出现在这里' : 'Deleted items will appear here'}
                </p>
              </div>
            ) : Array.from(groupedByDate.entries()).map(([dateKey, sessions]) => (
              <div key={dateKey} className="history-date-group">
                <div className="history-date-group__header">
                  <span className="history-date-group__date">{dateKey}</span>
                  <span className="history-date-group__count">{sessions.length}</span>
                </div>
                {sessions.map(item => (
                  <div
                    key={item.id}
                    className={`history-session-item history-session-item--deleted ${selectedId === item.id ? 'history-session-item--active' : ''}`}
                    onClick={() => setSelectedId(item.id)}
                  >
                    <div className="history-session-item__title">
                      <span style={{ marginRight: 6 }}>{typeIcon(item.type)}</span>
                      {item.name}
                    </div>
                    <div className="history-session-item__meta">
                      <span className="history-session-item__status">
                        ⏳{isZh ? '已删除' : 'Deleted'}
                      </span>
                      <span className="history-session-item__ws">
                        {typeName(item.type)}
                        {item.globalName && ` · ${item.globalName}`}
                      </span>
                      <span className="history-session-item__stats">
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
                <span>← {isZh ? '选择一个已删除的项目查看详情' : 'Select a deleted item to view details'}</span>
              </div>
            ) : (
              <>
                <div className="history-detail__header">
                  <div className="history-detail__title-row">
                    <h3 className="history-detail__title">
                      <span style={{ marginRight: 8 }}>{typeIcon(selectedItem.type)}</span>
                      {selectedItem.name}
                    </h3>
                  </div>
                  <div className="history-detail__actions">
                    <button className="history-btn history-btn--primary" onClick={() => handleRestore(selectedItem)}>
                      {isZh ? '恢复' : 'Restore'}
                    </button>
                    <button className="history-btn history-btn--danger" onClick={() => handleDeleteForever(selectedItem)}>
                      {isZh ? '彻底删除' : 'Delete Forever'}
                    </button>
                  </div>
                </div>

                <div className="history-detail__body">
                  <div className="history-detail__preview">
                    <div className="history-detail__meta-line">
                      <span className="history-detail__badge history-detail__badge--del">{typeName(selectedItem.type)}</span>
                      {selectedItem.globalName && <span className="history-detail__badge history-detail__badge--ws">{selectedItem.globalName}</span>}
                      {selectedItem.projectName && <span className="history-detail__badge history-detail__badge--ws">{selectedItem.projectName}</span>}
                      <span className="history-detail__time-del">{selectedItem.deletedAt}</span>
                    </div>

                    <div className="trash-content">
                      <table className="trash-table">
                        <thead>
                          <tr>
                            <th>{isZh ? '类型' : 'Type'}</th>
                            <th>{isZh ? '名称' : 'Name'}</th>
                            <th>{isZh ? '所属' : 'Parent'}</th>
                            <th>{isZh ? '删除时间' : 'Deleted At'}</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr>
                            <td>{typeName(selectedItem.type)}</td>
                            <td>{selectedItem.name}</td>
                            <td>{selectedItem.globalName || '-'}</td>
                            <td>{selectedItem.deletedAt}</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <div className="history-detail__toolbar">
                    <button className="history-toolbar-btn" onClick={() => handleRestore(selectedItem)} title={isZh ? '恢复' : 'Restore'}>
                      ↩️
                    </button>
                    <button className="history-toolbar-btn history-toolbar-btn--danger"
                      onClick={() => handleDeleteForever(selectedItem)} title={isZh ? '彻底删除' : 'Delete Forever'}>
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
