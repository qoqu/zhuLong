import { useEffect, useState, useCallback, useRef } from 'react'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { Transcript } from './components/Transcript'
import { Composer } from './components/Composer'
import { RightPanel } from './components/RightPanel'
import { StatusBar } from './components/StatusBar'
import { ApprovalModal } from './components/ApprovalModal'
import { SettingsPanel } from './components/SettingsPanel'
import { HistoryPage } from './components/HistoryPage'
import { RecycleBinPage, DeletedItem } from './components/RecycleBinPage'
import { Canvas } from './components/Canvas/Canvas'
import { exampleGoals } from './data/mock'
import type {
  AgentInfo,
  AgentStatus,
  ExecutionMode,
  Language,
  GlobalInfo,
  ProjectInfo,
  RightPanelTab,
  SessionState,
  SidebarView,
  Message,
  LogEntry,
  PlanStep,
  RuntimeStats,
  FileChange,
  ApprovalRequest,
  MemoryState,
  LearningState,
  ModuleState,
} from './types'
import { useT, resolveLanguage } from './i18n'
import './styles/canvas.css'

// Wails runtime — fall back to a browser-mode stub if not embedded.
const wails = typeof window !== 'undefined' && (window as any).runtime
const backend =
  typeof window !== 'undefined' && (window as any).go ? (window as any).go.main.App : null

const emptyStats = (model: string): RuntimeStats => ({
  totalUsed: 0,
  totalLimit: 1_000_000,
  usagePercent: 0,
  prompt: 0,
  completion: 0,
  reasoning: 0,
  other: 0,
  elapsed: '0s',
  requestCount: 0,
  sessionTokens: 0,
  cacheHitRatio: 0.8,
  mainCost: 0,
  mainCount: 0,
  subCost: 0,
  subCount: 0,
  balance: '¥73.23',
  currentSession: 0,
  sessionCost: '$0.0000',
  model,
  cacheHit: '未命中',
  avgHit: '未命中',
  thisTokens: 0,
  thisFee: '$0.0000',
  contextUsed: 0,
  compressPct: 0,
  remaining: '¥73.23',
})

function App() {
  // Theme
  const [darkMode, setDarkMode] = useState(true)
  const [language, setLanguage] = useState<Language>('zh')
  const [desktopStyle, setDesktopStyle] = useState<'classic' | 'workspace'>(() => {
    try { return (localStorage.getItem('zhulong-settings_desktopStyle') as 'classic' | 'workspace') || 'classic' }
    catch { return 'classic' }
  })

  // 侧边栏宽度状态（可拖拽调整）
  const [sidebarWidth, setSidebarWidth] = useState(280)
  const [rightPanelWidth, setRightPanelWidth] = useState(320)
  const dragStateRef = useRef<{ side: 'left' | 'right'; startX: number; startWidth: number } | null>(null)

  // 拖拽侧边栏宽度（全局 mousedown/move/up）
  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      const ds = dragStateRef.current
      if (!ds) return
      const dx = e.clientX - ds.startX
      if (ds.side === 'left') {
        const w = Math.max(180, Math.min(560, ds.startWidth + dx))
        setSidebarWidth(w)
      } else {
        const w = Math.max(220, Math.min(600, ds.startWidth - dx))
        setRightPanelWidth(w)
      }
    }
    const onUp = () => {
      dragStateRef.current = null
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [])

  const startDrag = (side: 'left' | 'right') => (e: React.MouseEvent) => {
    dragStateRef.current = {
      side,
      startX: e.clientX,
      startWidth: side === 'left' ? sidebarWidth : rightPanelWidth,
    }
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }

  // View mode
  const [viewMode, setViewMode] = useState<'chat' | 'canvas'>('chat')

  // Sidebar
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  // Right panel
  const [rightPanelCollapsed, setRightPanelCollapsed] = useState(false)
  const [sidebarView, setSidebarView] = useState<SidebarView>('projects')
  const [agents, setAgents] = useState<AgentInfo[]>([])
  const [globals, setGlobals] = useState<GlobalInfo[]>([])  // New: workspace list
  const [activeAgentId, setActiveAgentId] = useState('auto')
  const [activeGlobalId, setActiveGlobalId] = useState<string>('')  // New: current workspace
  const [activeProjectId, setActiveProjectId] = useState<string>('') // New: current project
  const [activeSessionId, setActiveSessionId] = useState<string>('')
  const activeSessionIdRef = useRef<string>('')

  // Modal states for creating Global/Project
  const [showCreateGlobalModal, setShowCreateGlobalModal] = useState(false)
  const [showCreateProjectModal, setShowCreateProjectModal] = useState(false)
  const [newGlobalName, setNewGlobalName] = useState('')
  const [newProjectName, setNewProjectName] = useState('')
  const [createProjectTargetGlobalId, setCreateProjectTargetGlobalId] = useState('')
  const [showSettings, setShowSettings] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [showTrash, setShowTrash] = useState(false)

  // Deleted items (recycle bin) — persisted in localStorage
  const TRASH_KEY = 'zhulong-trash-sessions'

  const [deletedSessions, setDeletedSessions] = useState<DeletedItem[]>(() => {
    try {
      const v = localStorage.getItem(TRASH_KEY)
      return v ? JSON.parse(v) : []
    } catch { return [] }
  })

  const saveTrash = (items: DeletedItem[]) => {
    try { localStorage.setItem(TRASH_KEY, JSON.stringify(items)) } catch {}
  }

  // Helper: find SessionInfo by id in globals tree
  const findSessionById = (gs: GlobalInfo[], sessionId: string): { session: any; globalName: string; projectName: string; globalId: string; projectId: string } | null => {
    for (const g of gs) {
      for (const p of g.projects) {
        const s = p.sessions.find((s: any) => s.id === sessionId)
        if (s) return { session: s, globalName: g.name, projectName: p.name, globalId: g.id, projectId: p.id }
      }
    }
    return null
  }

  // Active session runtime state
  const [status, setStatus] = useState<AgentStatus>('idle')
  const [executionMode, setExecutionMode] = useState<ExecutionMode>('auto')
  const [model, setModel] = useState('deepseek-v4-flash')
  const [temperature, setTemperature] = useState('auto')
  const [messages, setMessages] = useState<Message[]>([])
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [plan, setPlan] = useState<PlanStep[]>([])
  const [stats, setStats] = useState<RuntimeStats>(emptyStats('deepseek-v4-flash'))
  const [files, setFiles] = useState<string[]>([])
  const [changes, setChanges] = useState<FileChange[]>([])

  // Right panel
  const [rightTab, setRightTab] = useState<RightPanelTab>('overview')

  // New states for enhanced UI
  const [memoryState, setMemoryState] = useState<MemoryState | null>(null)
  const [learningState, setLearningState] = useState<LearningState | null>(null)
  const [moduleState, setModuleState] = useState<ModuleState | null>(null)

  // Composer input
  const [input, setInput] = useState('')

  // Approval modal
  const [approval, setApproval] = useState<ApprovalRequest | null>(null)


  // Helper: find Project by id in globals tree
  const findProjectById = (gs: GlobalInfo[], projectId: string): ProjectInfo | null => {
    for (const g of gs) {
      for (const p of g.projects) {
        if (p.id === projectId) return p
      }
    }
    return null
  }

  // ===== Initial load =====
  useEffect(() => {
    if (!backend) {
      // Browser fallback — load mocks
      import('./data/mock').then((m) => {
        setAgents(m.mockAgents)
        setGlobals(m.mockGlobals)
        // Activate first Global's first Project's first Session
        const firstGlobal = m.mockGlobals[0]
        if (firstGlobal) {
          setActiveGlobalId(firstGlobal.id)
          const firstProject = firstGlobal.projects[0]
          if (firstProject) {
            setActiveProjectId(firstProject.id)
            if (firstProject.sessions.length > 0) {
              setActiveSessionId(firstProject.sessions[0].id)
            }
          }
        }
      })
      return
    }
    backend
      .ListAgents()
      .then((a: AgentInfo[]) => setAgents(a))
      .catch(() => {})
    backend
      .ListGlobals()
      .then((g: GlobalInfo[]) => {
        setGlobals(g)
        // Activate first Global's first Project's first Session
        if (g.length > 0) {
          setActiveGlobalId(g[0].id)
          if (g[0].projects.length > 0) {
            setActiveProjectId(g[0].projects[0].id)
            if (g[0].projects[0].sessions.length > 0) {
              setActiveSessionId(g[0].projects[0].sessions[0].id)
            }
          }
        }
      })
      .catch(() => {})
  }, [])

  // Keep ref in sync with state
  useEffect(() => {
    activeSessionIdRef.current = activeSessionId
  }, [activeSessionId])

  // ===== Subscribe to backend events =====
  useEffect(() => {
    if (!wails) return
    const offSession = wails.EventsOn('session:update', (s: SessionState) => {
      if (!s || s.info?.id !== activeSessionIdRef.current) return
      applySession(s)
    })
    const offCreated = wails.EventsOn('session:created', (s: SessionState) => {
      if (!s) return
      setActiveSessionId(s.info.id)
      applySession(s)
    })
    const offDeleted = wails.EventsOn('session:deleted', (id: string) => {
      if (id === activeSessionIdRef.current) {
        // Switch to first remaining session in the same project
        setGlobals((currentGlobals) => {
          const project = currentGlobals[0]?.projects?.find((p) => p.id === activeProjectId)
          const first = project?.sessions?.find((x) => x.id !== id)
          if (first) setActiveSessionId(first.id)
          return currentGlobals
        })
      }
    })
    const offGlobals = wails.EventsOn('globals:update', (g: GlobalInfo[]) => {
      setGlobals(g)
    })
    return () => {
      offSession()
      offCreated()
      offDeleted()
      offGlobals()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeProjectId])

  // ===== When active session changes, pull its state =====
  useEffect(() => {
    if (!activeSessionId) return
    if (!backend) return
    backend
      .GetSession(activeSessionId)
      .then((s: SessionState | null) => {
        if (s) applySession(s)
      })
      .catch(() => {})
    backend.SetActiveSession(activeSessionId).catch(() => {})
  }, [activeSessionId])

  const applySession = (s: SessionState) => {
    setStatus(s.status as AgentStatus)
    setExecutionMode((s.mode as ExecutionMode) || 'auto')
    setModel(s.model || 'deepseek-v4-flash')
    setMessages(s.messages || [])
    setLogs(s.logs || [])
    setPlan(s.plan || [])
    setStats(s.stats || emptyStats(s.model || 'deepseek-v4-flash'))
    setFiles(s.files || [])
    setChanges(s.changes || [])
    
    // Update new states
    if (s.memoryState) {
      setMemoryState(s.memoryState)
    }
    if (s.learningState) {
      setLearningState(s.learningState)
    }
    if (s.moduleState) {
      setModuleState(s.moduleState)
    }
    
    if (s.approval) {
      setApproval(s.approval)
    } else {
      setApproval(null)
    }
  }

  // ===== Actions =====

  const handleSend = useCallback(
    async (text: string) => {
      if (!activeSessionId) return
      // Optimistic user message
      const userMsg: Message = {
        id: 'u' + Date.now(),
        role: 'user',
        content: text,
        time: new Date().toISOString(),
      }
      setMessages((m) => [...m, userMsg])
      setStatus('executing')
      setInput('')
      if (backend) {
        try {
          await backend.SendMessage(activeSessionId, text)
        } catch (e) {
          console.error(e)
        }
      } else {
        // Browser-mode demo: simulate
        setTimeout(() => {
          setPlan([
            { id: '1', description: '分析任务并收集上下文', status: 'completed' },
            { id: '2', description: '执行主要操作', status: 'running' },
            { id: '3', description: '验证并汇总结果', status: 'pending' },
          ])
          setLogs((l) => [
            ...l,
            { id: 'l1', time: new Date().toLocaleTimeString(), phase: 'plan', event: 'Plan created', detail: '3 steps' },
          ])
          // If in 'ask' mode, pop an approval modal mid-run
          if (executionMode === 'ask') {
            setTimeout(() => {
              setStatus('waiting_human')
              setApproval({
                id: 'ap' + Date.now(),
                tool: 'execute_command',
                args: { cmd: 'rm -rf ./build' },
                risk: 'high',
                reason: '该命令会删除 build 目录及其所有内容，且不可恢复。',
                createdAt: new Date().toISOString(),
              })
            }, 600)
            return
          }
          setTimeout(() => {
            setPlan((p) =>
              p.map((s) =>
                s.id === '2' ? { ...s, status: 'completed' } : s.id === '3' ? { ...s, status: 'running' } : s
              )
            )
            setMessages((m) => [
              ...m,
              {
                id: 'a' + Date.now(),
                role: 'assistant',
                content: '任务完成。我已经分析并执行了你的请求，所有步骤都成功了。',
                time: new Date().toISOString(),
              },
            ])
            setStatus('done')
            setStats((s) => ({
              ...s,
              sessionTokens: s.sessionTokens + 2100,
              requestCount: s.requestCount + 2,
              mainCount: s.mainCount + 2,
              mainCost: s.mainCost + 0.007,
              elapsed: '2.4s',
            }))
          }, 1200)
        }, 600)
      }
    },
    [activeSessionId, executionMode]
  )

  const handleStop = useCallback(async () => {
    if (backend) {
      try {
        await backend.Stop()
      } catch {}
    }
    setStatus('idle')
  }, [])

  const handleReset = useCallback(async () => {
    if (backend) {
      try {
        await backend.Reset()
      } catch {}
    }
    setMessages([])
    setLogs([])
    setPlan([])
    setStats(emptyStats(model))
    setStatus('idle')
  }, [model])

  const handleNewSession = useCallback(async (projectId?: string) => {
    let targetProjectId = projectId || activeProjectId || globals[0]?.projects[0]?.id || ''

    // 自动创建：如果没有工作空间和项目，依次创建
    if (!targetProjectId && backend) {
      try {
        // 1. 创建工作空间
        const globalName = resolveLanguage(language) === 'zh' ? '新的工作空间' : 'New Workspace'
        await backend.CreateGlobal(globalName)

        // 2. 重新加载 globals 获取新数据
        const updatedGlobals = await backend.ListGlobals()
        setGlobals(updatedGlobals)

        // 3. 获取最新创建的工作空间和项目
        const latestGlobal = updatedGlobals[updatedGlobals.length - 1]
        if (latestGlobal && latestGlobal.projects.length === 0) {
          const projectName = resolveLanguage(language) === 'zh' ? '新的工作区' : 'New Project'
          await backend.CreateProject(latestGlobal.id, projectName)
          // 重新加载
          const g2 = await backend.ListGlobals()
          setGlobals(g2)
          const latest = g2[g2.length - 1]
          targetProjectId = latest?.projects[0]?.id || ''
        } else if (latestGlobal) {
          targetProjectId = latestGlobal.projects[0]?.id || ''
        }
      } catch (e) { console.error('Auto-create failed:', e) }
    }

    // 浏览器模式：本地创建
    if (!targetProjectId && !backend) {
      const globalId = 'g-' + (crypto.randomUUID ? crypto.randomUUID() : Date.now().toString())
      const pid = 'p-' + (crypto.randomUUID ? crypto.randomUUID() : Date.now().toString())
      const globalName = resolveLanguage(language) === 'zh' ? '新的工作空间' : 'New Workspace'
      const projectName = resolveLanguage(language) === 'zh' ? '新的工作区' : 'New Project'
      setGlobals([{
        id: globalId,
        name: globalName,
        projects: [{
          id: pid,
          name: projectName,
          sessions: [],
        }],
      }])
      setActiveGlobalId(globalId)
      targetProjectId = pid
    }

    if (!targetProjectId) return
    if (backend) {
      try {
        const result = await backend.NewSession(targetProjectId) as any
        const newId = result?.info?.id
        if (newId) setActiveSessionId(newId)
        setMessages([])
        setLogs([])
        setPlan([])
        setStatus('idle')
        return
      } catch (e) { console.error('NewSession failed:', e) }
    }
    // Browser fallback
    const id = 's-' + (crypto.randomUUID ? crypto.randomUUID() : Date.now() + '-' + Math.random().toString(36).slice(2))
    setGlobals((gs) => {
      return gs.map(g => ({
        ...g,
        projects: g.projects.map(p =>
          p.id === targetProjectId
            ? {
                ...p,
                sessions: [
                  { id, title: '新会话', agentId: activeAgentId, projectId: targetProjectId, messageCount: 0, toolCount: 0, updatedAt: '刚刚', preview: '新会话' },
                  ...p.sessions,
                ],
              }
            : p
        ),
      }))
    })
    setActiveSessionId(id)
    setMessages([])
    setLogs([])
    setPlan([])
    setStatus('idle')
  }, [activeProjectId, activeAgentId, backend, globals, language])

  const handleDeleteSession = useCallback(
    async (id: string) => {
      // ── Before deleting, save session info to recycle bin ──
      const found = findSessionById(globals, id)
      if (found) {
        const now = new Date()
        const dateKey = `${now.getFullYear()}/${now.getMonth() + 1}/${now.getDate()}`
        const trashEntry: DeletedItem = {
          id: id,
          type: 'session',
          name: found.session.title || '新会话',
          globalName: found.globalName,
          projectName: found.projectName,
          globalId: found.globalId,
          projectId: found.projectId,
          dateKey,
          timestamp: now.getTime(),
          messageCount: found.session.messageCount || 0,
          toolCount: found.session.toolCount || 0,
          preview: found.session.preview || '',
          deletedAt: `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`,
        }
        setDeletedSessions(prev => {
          const next = [trashEntry, ...prev]
          saveTrash(next)
          return next
        })
      }

      // ── Then actually delete from backend or local state ──
      if (backend) {
        try {
          await backend.DeleteSession(id)
          return
        } catch {}
      }
      // Browser fallback: remove from globals tree (deep copy for React immutability)
      setGlobals((gs) => gs.map(g => ({
        ...g,
        projects: g.projects.map(p => ({
          ...p,
          sessions: p.sessions.filter((s) => s.id !== id)
        }))
      })))
      if (id === activeSessionId) {
        // Switch to first remaining session in same project
        const project = findProjectById(globals, activeProjectId)
        const first = project?.sessions?.find((x) => x.id !== id)
        if (first) setActiveSessionId(first.id)
      }
    },
    [activeSessionId, activeProjectId, globals]
  )

  const handleRenameSession = useCallback((id: string) => {
    // Find current title from globals tree
    let curTitle = ''
    for (const g of globals) {
      for (const p of g.projects) {
        const s = p.sessions.find((s) => s.id === id)
        if (s) { curTitle = s.title; break }
      }
    }
    if (!curTitle) return
    const next = window.prompt(resolveLanguage(language) === 'zh' ? `重命名「${curTitle}」：` : `Rename "${curTitle}":`, curTitle)
    if (next && next.trim()) {
      if (backend) backend.RenameSession(id, next.trim())
      // Update in globals tree (deep copy for React immutability)
      setGlobals((gs) => gs.map(g => ({
        ...g,
        projects: g.projects.map(p => ({
          ...p,
          sessions: p.sessions.map(s =>
            s.id === id ? { ...s, title: next.trim(), preview: next.trim() } : s
          )
        }))
      })))
    }
  }, [language, globals])

  const handleRestoreItem = useCallback(async (item: DeletedItem) => {
    // 调用后端恢复
    if (backend) {
      try {
        await backend.RestoreFromRecycleBin(item.id, item.type)
        // 后端会更新 globals 树，重新加载
        const g = await backend.ListGlobals()
        setGlobals(g)
        return
      } catch {}
    }

    // 浏览器模式恢复
    setDeletedSessions(prev => {
      const next = prev.filter(t => t.id !== item.id)
      saveTrash(next)
      return next
    })

    if (item.type === 'session') {
      setGlobals(gs => gs.map(g =>
        g.id === item.globalId ? {
          ...g,
          projects: g.projects.map(p =>
            p.id === item.projectId ? {
              ...p,
              sessions: [{
                id: item.id,
                title: item.name,
                agentId: 'auto',
                projectId: item.projectId,
                messageCount: item.messageCount || 0,
                toolCount: item.toolCount || 0,
                updatedAt: item.deletedAt,
                preview: item.preview || '',
              }, ...p.sessions],
            } : p
          )
        } : g
      ))
    } else if (item.type === 'project') {
      setGlobals(gs => gs.map(g =>
        g.id === item.globalId ? {
          ...g,
          projects: [...g.projects, { id: item.id, name: item.name, sessions: [] }]
        } : g
      ))
    } else if (item.type === 'global') {
      setGlobals(gs => [...gs, { id: item.id, name: item.name, projects: [] }])
    }
  }, [backend, deletedSessions])

  const handleCreateGlobal = useCallback(() => {
    setNewGlobalName(resolveLanguage(language) === 'zh' ? '新的工作空间' : 'New Workspace')
    setShowCreateGlobalModal(true)
  }, [language])

  const handleCreateProject = useCallback((globalId: string) => {
    setNewProjectName(resolveLanguage(language) === 'zh' ? '新的工作区' : 'New Project')
    setCreateProjectTargetGlobalId(globalId)
    setShowCreateProjectModal(true)
  }, [language])

  const handleConfirmCreateGlobal = useCallback(async () => {
    const name = newGlobalName.trim()
    if (!name) return
    setShowCreateGlobalModal(false)
    if (backend) {
      try {
        await backend.CreateGlobal(name)
        // Don't manually add — backend will emit 'globals:update' event
      } catch (e: any) {
        alert(resolveLanguage(language) === 'zh' ? `创建失败: ${e?.message || e}` : `Failed: ${e?.message || e}`)
      }
    } else {
      // Browser fallback
      const id = 'g-' + (crypto.randomUUID ? crypto.randomUUID() : Date.now() + '-' + Math.random().toString(36).slice(2))
      const newGlobal: GlobalInfo = { id, name, projects: [] }
      setGlobals((gs) => [...gs, newGlobal])
      setActiveGlobalId(id)
    }
  }, [newGlobalName, language])

  const handleConfirmCreateProject = useCallback(async () => {
    const name = newProjectName.trim()
    if (!name) return
    setShowCreateProjectModal(false)
    if (backend) {
      try {
        await backend.CreateProject(createProjectTargetGlobalId, name)
        // Don't manually add — backend will emit 'globals:update' event
      } catch (e: any) {
        alert(resolveLanguage(language) === 'zh' ? `创建失败: ${e?.message || e}` : `Failed: ${e?.message || e}`)
      }
    } else {
      // Browser fallback
      const id = 'p-' + (crypto.randomUUID ? crypto.randomUUID() : Date.now() + '-' + Math.random().toString(36).slice(2))
      const newProject: ProjectInfo = { id, name, sessions: [] }
      setGlobals((gs) => gs.map(g =>
        g.id === createProjectTargetGlobalId
          ? { ...g, projects: [...g.projects, newProject] }
          : g
      ))
      setActiveProjectId(id)
    }
  }, [newProjectName, createProjectTargetGlobalId, language])

  // ── 删除工作空间 ──
  const handleDeleteGlobal = useCallback(async (globalId: string) => {
    const ws = globals.find(g => g.id === globalId)
    const name = ws?.name || globalId
    const isZh = resolveLanguage(language) === 'zh'
    const msg = isZh ? `确定要删除工作空间「${name}」吗？所有项目和对话将移入回收站。` : `Delete workspace "${name}"? All projects and sessions will be moved to recycle bin.`
    if (!window.confirm(msg)) return

    // 添加到回收站
    const now = new Date()
    const trashEntry: DeletedItem = {
      id: globalId,
      type: 'global',
      name: name,
      dateKey: `${now.getFullYear()}/${now.getMonth() + 1}/${now.getDate()}`,
      timestamp: now.getTime(),
      deletedAt: `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`,
    }
    setDeletedSessions(prev => {
      const next = [trashEntry, ...prev]
      saveTrash(next)
      return next
    })

    if (backend) {
      try {
        await backend.DeleteGlobal(globalId)
        const g = await backend.ListGlobals()
        setGlobals(g)
        return
      } catch {}
    }
    // 浏览器模式
    setGlobals(gs => gs.filter(g => g.id !== globalId))
  }, [globals, language, backend])

  // ── 删除项目 ──
  const handleDeleteProject = useCallback(async (globalId: string, projectId: string) => {
    const ws = globals.find(g => g.id === globalId)
    const proj = ws?.projects.find(p => p.id === projectId)
    const name = proj?.name || projectId
    const isZh = resolveLanguage(language) === 'zh'
    const msg = isZh ? `确定要删除项目「${name}」吗？所有对话将移入回收站。` : `Delete project "${name}"? All sessions will be moved to recycle bin.`
    if (!window.confirm(msg)) return

    // 添加到回收站
    const now = new Date()
    const trashEntry: DeletedItem = {
      id: projectId,
      type: 'project',
      name: name,
      globalId: globalId,
      globalName: ws?.name || '',
      dateKey: `${now.getFullYear()}/${now.getMonth() + 1}/${now.getDate()}`,
      timestamp: now.getTime(),
      deletedAt: `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`,
    }
    setDeletedSessions(prev => {
      const next = [trashEntry, ...prev]
      saveTrash(next)
      return next
    })

    if (backend) {
      try {
        await backend.DeleteProject(globalId, projectId)
        const g = await backend.ListGlobals()
        setGlobals(g)
        return
      } catch {}
    }
    // 浏览器模式
    setGlobals(gs => gs.map(g => ({
      ...g,
      projects: g.projects.filter(p => p.id !== projectId)
    })))
  }, [globals, language, backend])

  const handleSelectAgent = useCallback(
    async (id: string) => {
      setActiveAgentId(id)
      if (backend) {
        try {
          await backend.SetActiveAgent(id)
        } catch {}
      }
    },
    []
  )

  const handleChangeMode = useCallback(async (m: ExecutionMode) => {
    setExecutionMode(m)
    if (backend) {
      try {
        await backend.SetExecutionMode(m)
      } catch {}
    }
  }, [])

  const handleChangeModel = useCallback(async (m: string) => {
    setModel(m)
    if (backend) {
      try {
        await backend.SetModel(m)
      } catch {}
    }
  }, [])

  const handleChangeTemperature = useCallback(async (t: string) => {
    setTemperature(t)
    if (backend) {
      try {
        await backend.SetTemperature(t)
      } catch {}
    }
  }, [])

  return (
    <div
      className={`app ${darkMode ? 'theme--dark' : 'theme--light'}`}
      style={{
        gridTemplateColumns: `${sidebarCollapsed ? 0 : sidebarWidth}px ${sidebarCollapsed ? 0 : 6}px 1fr 6px ${rightPanelCollapsed ? 0 : rightPanelWidth}px`,
      }}
    >
      <Sidebar
        language={language}
        onLanguageChange={setLanguage}
        darkMode={darkMode}
        onToggleDarkMode={() => setDarkMode((v) => !v)}
        collapsed={sidebarCollapsed}
        onToggleCollapsed={() => setSidebarCollapsed((v) => !v)}
        view={sidebarView}
        onChangeView={setSidebarView}
        onOpenSettings={() => setShowSettings(true)}
        desktopStyle={desktopStyle}
        executionMode={executionMode}
        agents={agents}
        globals={globals}
        activeGlobalId={activeGlobalId}
        activeProjectId={activeProjectId}
        activeAgentId={activeAgentId}
        onSelectAgent={handleSelectAgent}
        activeSessionId={activeSessionId}
        onSelectSession={setActiveSessionId}
        onSelectGlobal={setActiveGlobalId}
        onSelectProject={setActiveProjectId}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
        onRenameSession={handleRenameSession}
        onDeleteGlobal={handleDeleteGlobal}
        onDeleteProject={handleDeleteProject}
        onCreateGlobal={handleCreateGlobal}
        onCreateProject={handleCreateProject}
        onOpenHistory={() => setShowHistory(true)}
        onOpenTrash={() => setShowTrash(true)}
      />

      <div
        className="resizer resizer--left"
        onMouseDown={startDrag('left')}
        title="拖拽调整侧边栏宽度"
      />

      <main className="main">
        <TopBar
          language={language}
          sessions={globals.flatMap((g) => g.projects.flatMap((p) => p.sessions || [])).map(s => ({ id: s.id, title: s.title }))}
          activeSessionId={activeSessionId}
          onSwitchSession={(id) => {
            setActiveSessionId(id)
          }}
          onCloseSession={handleDeleteSession}
          onToggleSidebar={() => setSidebarCollapsed((v) => !v)}
          onToggleRightPanel={() => setRightPanelCollapsed((v) => !v)}
        />

        {/* 视图切换按钮 */}
        <div
          style={{
            display: 'flex',
            gap: 'var(--space-2)',
            padding: 'var(--space-2) var(--space-4)',
            borderBottom: '1px solid var(--border)',
            background: 'var(--bg-soft)',
          }}
        >
          <button
            onClick={() => setViewMode('chat')}
            style={{
              padding: 'var(--space-2) var(--space-3)',
              background: viewMode === 'chat' ? 'var(--accent-soft)' : 'transparent',
              color: viewMode === 'chat' ? 'var(--accent)' : 'var(--fg-faint)',
              border: 'none',
              borderRadius: 'var(--radius)',
              fontSize: 'var(--text-sm)',
              fontWeight: 500,
              cursor: 'pointer',
              transition: 'all 0.2s ease',
            }}
          >
            💬 Chat
          </button>
          <button
            onClick={() => setViewMode('canvas')}
            style={{
              padding: 'var(--space-2) var(--space-3)',
              background: viewMode === 'canvas' ? 'var(--accent-soft)' : 'transparent',
              color: viewMode === 'canvas' ? 'var(--accent)' : 'var(--fg-faint)',
              border: 'none',
              borderRadius: 'var(--radius)',
              fontSize: 'var(--text-sm)',
              fontWeight: 500,
              cursor: 'pointer',
              transition: 'all 0.2s ease',
            }}
          >
            🎨 Canvas
          </button>
        </div>

        {/* 根据视图模式显示不同内容 */}
        {viewMode === 'chat' ? (
          <>
            {messages.length === 0 ? (
              <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
                <EmptyState
                  language={language}
                  onPick={(text) => setInput(text)}
                />
              </div>
            ) : (
              <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
                <Transcript
                  language={language}
                  messages={messages}
                  logs={logs}
                  plan={plan}
                  status={status}
                />
              </div>
            )}

            <div className="main__composer">
              <Composer
                language={language}
                status={status}
                executionMode={executionMode}
                onChangeExecutionMode={handleChangeMode}
                model={model}
                onChangeModel={handleChangeModel}
                temperature={temperature}
                onChangeTemperature={handleChangeTemperature}
                onSend={handleSend}
                onStop={handleStop}
                onReset={handleReset}
                value={input}
                onChange={setInput}
                waitingHuman={status === 'waiting_human'}
              />
            </div>
          </>
        ) : (
          <div style={{ flex: 1, position: 'relative' }}>
            <Canvas />
          </div>
        )}
      </main>

      {!rightPanelCollapsed && (
        <>
          <div
            className="resizer resizer--right"
            onMouseDown={startDrag('right')}
            title="拖拽调整右侧面板宽度"
          />

          <RightPanel
            language={language}
            tab={rightTab}
            onChangeTab={setRightTab}
            stats={stats}
            files={files}
            changes={changes}
            memoryState={memoryState}
            learningState={learningState}
            moduleState={moduleState}
          />
        </>
      )}

      <StatusBar
        language={language}
        status={status}
        model={model}
        stats={stats}
        executionMode={executionMode}
      />

      {approval && (
        <ApprovalModal
          language={language}
          request={approval}
          onApprove={async () => {
            const id = approval.id
            setApproval(null)
            if (backend) {
              try {
                await backend.RespondApproval(id, true)
              } catch {}
            }
          }}
          onDeny={async () => {
            const id = approval.id
            setApproval(null)
            if (backend) {
              try {
                await backend.RespondApproval(id, false)
              } catch {}
            }
          }}
          onAlwaysAllow={async () => {
            const id = approval.id
            setApproval(null)
            if (backend) {
              try {
                await backend.RespondApproval(id, true)
              } catch {}
            }
            setExecutionMode('yolo')
          }}
        />
      )}

      {/* Create Global Modal */}
      {showCreateGlobalModal && (
        <div className="modal-overlay" onClick={() => setShowCreateGlobalModal(false)}>
          <div className="modal-dialog" onClick={(e) => e.stopPropagation()}>
            <div className="modal-dialog__header">
              <span className="modal-dialog__title">{resolveLanguage(language) === 'zh' ? '创建新工作空间' : 'Create New Workspace'}</span>
              <button className="modal-dialog__close" onClick={() => setShowCreateGlobalModal(false)}>✕</button>
            </div>
            <div className="modal-dialog__body">
              <input
                autoFocus
                className="modal-dialog__input"
                value={newGlobalName}
                onChange={(e) => setNewGlobalName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleConfirmCreateGlobal()}
                placeholder={resolveLanguage(language) === 'zh' ? '输入工作空间名称' : 'Enter workspace name'}
              />
            </div>
            <div className="modal-dialog__footer">
              <button className="modal-dialog__btn modal-dialog__btn--cancel" onClick={() => setShowCreateGlobalModal(false)}>
                {resolveLanguage(language) === 'zh' ? '取消' : 'Cancel'}
              </button>
              <button className="modal-dialog__btn modal-dialog__btn--primary" onClick={handleConfirmCreateGlobal}>
                {resolveLanguage(language) === 'zh' ? '创建' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Project Modal */}
      {showCreateProjectModal && (
        <div className="modal-overlay" onClick={() => setShowCreateProjectModal(false)}>
          <div className="modal-dialog" onClick={(e) => e.stopPropagation()}>
            <div className="modal-dialog__header">
              <span className="modal-dialog__title">{resolveLanguage(language) === 'zh' ? '创建新工作区' : 'Create New Project'}</span>
              <button className="modal-dialog__close" onClick={() => setShowCreateProjectModal(false)}>✕</button>
            </div>
            <div className="modal-dialog__body">
              <input
                autoFocus
                className="modal-dialog__input"
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleConfirmCreateProject()}
                placeholder={resolveLanguage(language) === 'zh' ? '输入工作区名称' : 'Enter project name'}
              />
            </div>
            <div className="modal-dialog__footer">
              <button className="modal-dialog__btn modal-dialog__btn--cancel" onClick={() => setShowCreateProjectModal(false)}>
                {resolveLanguage(language) === 'zh' ? '取消' : 'Cancel'}
              </button>
              <button className="modal-dialog__btn modal-dialog__btn--primary" onClick={handleConfirmCreateProject}>
                {resolveLanguage(language) === 'zh' ? '创建' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Settings Modal */}
      {showSettings && (
        <SettingsPanel
          language={language}
          darkMode={darkMode}
          executionMode={executionMode}
          model={model}
          desktopStyle={desktopStyle}
          onLanguageChange={setLanguage}
          onToggleDarkMode={() => setDarkMode((v) => !v)}
          onModelChange={setModel}
          onExecutionModeChange={setExecutionMode}
          onDesktopStyleChange={setDesktopStyle}
          onClose={() => setShowSettings(false)}
        />
      )}

      {/* History Page */}
      {showHistory && (
        <HistoryPage
          language={language}
          globals={globals}
          activeSessionId={activeSessionId}
          onClose={() => setShowHistory(false)}
          onSelectSession={(id) => setActiveSessionId(id)}
          onOpenAll={() => { /* already handled in onSelectSession */ }}
          onMoveToTrash={(id) => handleDeleteSession(id)}
          onRename={(id) => handleRenameSession(id)}
        />
      )}

      {/* Recycle Bin */}
      {showTrash && (
        <RecycleBinPage
          language={language}
          globals={globals}
          deletedSessions={deletedSessions}
          onClose={() => setShowTrash(false)}
          onRestore={handleRestoreItem}
        />
      )}
    </div>
  )
}

function EmptyState(props: { language: Language; onPick: (text: string) => void }) {
  const t = useT(props.language)
  const resolvedLang = resolveLanguage(props.language)
  return (
    <div className="transcript__empty">
      <div className="transcript__empty-icon">Z</div>
      <h2 className="transcript__empty-title">{t.emptyTitle}</h2>
      <p className="transcript__empty-desc">{t.emptyDesc}</p>
      <div className="transcript__examples">
        {exampleGoals.map((eg, i) => (
          <button
            key={i}
            className="transcript__example-chip"
            onClick={() => props.onPick(eg.text[resolvedLang])}
          >
            <span>{eg.icon}</span>
            <span>{eg.text[resolvedLang]}</span>
          </button>
        ))}
      </div>
    </div>
  )
}

export default App
