import { useEffect, useState, useCallback } from 'react'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { Transcript } from './components/Transcript'
import { Composer } from './components/Composer'
import { RightPanel } from './components/RightPanel'
import { StatusBar } from './components/StatusBar'
import { ApprovalModal } from './components/ApprovalModal'
import { Canvas } from './components/Canvas/Canvas'
import { exampleGoals } from './data/mock'
import type {
  AgentInfo,
  AgentStatus,
  ExecutionMode,
  InputMode,
  Language,
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
import { useT } from './i18n'
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

  // View mode
  const [viewMode, setViewMode] = useState<'chat' | 'canvas'>('chat')

  // Sidebar
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [sidebarView, setSidebarView] = useState<SidebarView>('projects')
  const [agents, setAgents] = useState<AgentInfo[]>([])
  const [projects, setProjects] = useState<ProjectInfo[]>([])
  const [activeAgentId, setActiveAgentId] = useState('auto')
  const [activeSessionId, setActiveSessionId] = useState<string>('')

  // Active session runtime state
  const [status, setStatus] = useState<AgentStatus>('idle')
  const [executionMode, setExecutionMode] = useState<ExecutionMode>('auto')
  const [inputMode, setInputMode] = useState<InputMode>('normal')
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

  const t = useT(language)

  // ===== Initial load =====
  useEffect(() => {
    if (!backend) {
      // Browser fallback — load mocks
      import('./data/mock').then((m) => {
        setAgents(m.mockAgents)
        setProjects(m.mockProjects)
        setActiveSessionId('s2')
      })
      return
    }
    backend
      .ListAgents()
      .then((a: AgentInfo[]) => setAgents(a))
      .catch(() => {})
    backend
      .ListProjects()
      .then((p: ProjectInfo[]) => {
        setProjects(p)
        const first =
          p[0]?.sessions?.[0]?.id || ''
        if (first) setActiveSessionId(first)
      })
      .catch(() => {})
  }, [])

  // ===== Subscribe to backend events =====
  useEffect(() => {
    if (!wails) return
    const offSession = wails.EventsOn('session:update', (s: SessionState) => {
      if (!s || s.info?.id !== activeSessionId) return
      applySession(s)
    })
    const offCreated = wails.EventsOn('session:created', (s: SessionState) => {
      if (!s) return
      setActiveSessionId(s.info.id)
      applySession(s)
    })
    const offDeleted = wails.EventsOn('session:deleted', (id: string) => {
      if (id === activeSessionId) {
        // Switch to first remaining
        const first = projects[0]?.sessions?.find((x) => x.id !== id)
        if (first) setActiveSessionId(first.id)
      }
    })
    const offProjects = wails.EventsOn('projects:update', (p: ProjectInfo[]) => {
      setProjects(p)
    })
    return () => {
      offSession()
      offCreated()
      offDeleted()
      offProjects()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeSessionId, projects])

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

  const handleNewSession = useCallback(async () => {
    if (backend) {
      try {
        await backend.NewSession()
        return
      } catch {}
    }
    // Browser fallback
    const id = 's' + Date.now()
    setProjects((p) => {
      if (!p[0]) return p
      return [
        {
          ...p[0],
          sessions: [
            { id, title: '新会话', agentId: activeAgentId, projectId: 'global', messageCount: 0, toolCount: 0, updatedAt: '刚刚', preview: '新会话' },
            ...p[0].sessions,
          ],
        },
        ...p.slice(1),
      ]
    })
    setActiveSessionId(id)
    setMessages([])
    setLogs([])
    setPlan([])
    setStatus('idle')
  }, [activeAgentId])

  const handleDeleteSession = useCallback(
    async (id: string) => {
      if (backend) {
        try {
          await backend.DeleteSession(id)
          return
        } catch {}
      }
      setProjects((p) => {
        if (!p[0]) return p
        return [
          { ...p[0], sessions: p[0].sessions.filter((s) => s.id !== id) },
          ...p.slice(1),
        ]
      })
      if (id === activeSessionId) {
        const first = projects[0]?.sessions?.find((x) => x.id !== id)
        if (first) setActiveSessionId(first.id)
      }
    },
    [activeSessionId, projects]
  )

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

  return (
    <div className={`app ${darkMode ? 'theme--dark' : 'theme--light'}`}>
      <Sidebar
        language={language}
        onLanguageChange={setLanguage}
        collapsed={sidebarCollapsed}
        onToggleCollapsed={() => setSidebarCollapsed((v) => !v)}
        view={sidebarView}
        onChangeView={setSidebarView}
        agents={agents}
        projects={projects}
        activeAgentId={activeAgentId}
        onSelectAgent={handleSelectAgent}
        activeSessionId={activeSessionId}
        onSelectSession={setActiveSessionId}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
      />

      <main className="main">
        <TopBar
          language={language}
          sessionTitle={
            projects
              .flatMap((p) => p.sessions || [])
              .find((s) => s.id === activeSessionId)?.title || t.untitled
          }
          sessionScope="Global"
          onRename={() => {
            const cur = projects
              .flatMap((p) => p.sessions || [])
              .find((s) => s.id === activeSessionId)
            if (!cur) return
            const next = window.prompt(t.renamePrompt, cur.title)
            if (next && next.trim()) {
              if (backend) backend.RenameSession(activeSessionId, next.trim())
              setProjects((p) =>
                p.map((proj) => ({
                  ...proj,
                  sessions: proj.sessions.map((s) =>
                    s.id === activeSessionId ? { ...s, title: next.trim(), preview: next.trim() } : s
                  ),
                }))
              )
            }
          }}
          onExport={() => {
            const text = messages
              .map((m) => `[${m.role}] ${m.content}`)
              .join('\n\n')
            const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
            const url = URL.createObjectURL(blob)
            const a = document.createElement('a')
            a.href = url
            a.download = `session-${activeSessionId}.txt`
            a.click()
            URL.revokeObjectURL(url)
          }}
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
              <EmptyState
                language={language}
                onPick={(text) => setInput(text)}
              />
            ) : (
              <Transcript
                language={language}
                messages={messages}
                logs={logs}
                plan={plan}
                status={status}
              />
            )}

            <div className="main__composer">
              <Composer
                language={language}
                status={status}
                executionMode={executionMode}
                onChangeExecutionMode={handleChangeMode}
                inputMode={inputMode}
                onChangeInputMode={setInputMode}
                model={model}
                onChangeModel={handleChangeModel}
                temperature={temperature}
                onChangeTemperature={setTemperature}
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

      <StatusBar
        language={language}
        status={status}
        model={model}
        stats={stats}
        darkMode={darkMode}
        onToggleDarkMode={() => setDarkMode((v) => !v)}
        onToggleLanguage={() => setLanguage((l) => (l === 'en' ? 'zh' : 'en'))}
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
    </div>
  )
}

function EmptyState(props: { language: Language; onPick: (text: string) => void }) {
  const t = useT(props.language)
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
            onClick={() => props.onPick(eg.text[props.language])}
          >
            <span>{eg.icon}</span>
            <span>{eg.text[props.language]}</span>
          </button>
        ))}
      </div>
    </div>
  )
}

export default App
