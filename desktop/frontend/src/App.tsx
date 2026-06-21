import { useState, useRef, useEffect } from 'react'

// Types
interface Message {
  id: string
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  timestamp: Date
  toolName?: string
  toolCalls?: ToolCall[]
}

interface ToolCall {
  id: string
  name: string
  args: string
  result?: string
  status: 'pending' | 'running' | 'completed' | 'failed'
}

interface LogEntry {
  id: string
  time: string
  phase: string
  event: string
  detail?: string
}

interface PlanStep {
  id: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed'
}

type ExecutionMode = 'ask' | 'auto' | 'yolo'
type RightPanel = 'logs' | 'plan' | 'context' | 'settings'
type SidebarView = 'chat' | 'history' | 'agents'

function App() {
  // State
  const [goal, setGoal] = useState('')
  const [status, setStatus] = useState<'idle' | 'planning' | 'executing' | 'reflecting' | 'replanning' | 'waiting_human' | 'done' | 'error'>('idle')
  const [messages, setMessages] = useState<Message[]>([])
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [plan, setPlan] = useState<PlanStep[]>([])
  const [darkMode, setDarkMode] = useState(true)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [sidebarView, setSidebarView] = useState<SidebarView>('chat')
  const [rightPanel, setRightPanel] = useState<RightPanel>('logs')
  const [executionMode, setExecutionMode] = useState<ExecutionMode>('auto')
  const [inputValue, setInputValue] = useState('')
  
  // Stats
  const [loops, setLoops] = useState(0)
  const [tokens, setTokens] = useState(0)
  const [cost, setCost] = useState(0)
  const [duration, setDuration] = useState('0s')
  
  // Refs
  const transcriptRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Auto-scroll transcript
  useEffect(() => {
    if (transcriptRef.current) {
      transcriptRef.current.scrollTop = transcriptRef.current.scrollHeight
    }
  }, [messages])

  // Helper functions
  const addLog = (phase: string, event: string, detail?: string) => {
    setLogs(prev => [...prev, {
      id: Date.now().toString(),
      time: new Date().toLocaleTimeString(),
      phase,
      event,
      detail
    }])
  }

  const addMessage = (role: Message['role'], content: string, toolName?: string) => {
    setMessages(prev => [...prev, {
      id: Date.now().toString(),
      role,
      content,
      timestamp: new Date(),
      toolName
    }])
  }

  // Actions
  const handleStart = () => {
    if (!goal.trim()) return
    
    setStatus('planning')
    setMessages([{
      id: Date.now().toString(),
      role: 'user',
      content: goal,
      timestamp: new Date()
    }])
    setLogs([])
    setPlan([])
    setLoops(0)
    setTokens(0)
    setCost(0)
    addLog('system', 'Agent started', `Goal: ${goal}`)
    addLog('system', `Execution mode: ${executionMode}`)

    // Simulate agent loop
    setTimeout(() => {
      setStatus('executing')
      addLog('planning', 'Plan created', '3 steps')
      setPlan([
        { id: '1', description: 'Analyze the task and gather information', status: 'completed' },
        { id: '2', description: 'Execute the main action', status: 'running' },
        { id: '3', description: 'Verify and summarize results', status: 'pending' }
      ])
      addMessage('assistant', 'I\'ll help you achieve this goal. Let me start by analyzing the task...')
    }, 1500)

    setTimeout(() => {
      addLog('executing', 'Step 1/3 completed', 'read_file: main.go')
      addMessage('tool', 'Reading file: main.go', 'read_file')
      setLoops(1)
      setTokens(1234)
      setCost(0.02)
    }, 3000)

    setTimeout(() => {
      setStatus('reflecting')
      addLog('executing', 'Step 2/3 completed', 'execute_command: go build')
      addMessage('tool', 'Executing: go build ./...', 'execute_command')
      setPlan(prev => prev.map(s => s.id === '2' ? { ...s, status: 'completed' } : s))
      setTokens(2500)
      setCost(0.04)
    }, 4500)

    setTimeout(() => {
      setStatus('done')
      addLog('reflecting', 'Goal achieved')
      addMessage('assistant', 'Task completed successfully! Here\'s what I found:\n\n1. The codebase compiles without errors\n2. All tests pass\n3. No security issues detected')
      setPlan(prev => prev.map(s => s.id === '3' ? { ...s, status: 'completed' } : s))
      setDuration('6.2s')
      setTokens(3800)
      setCost(0.06)
    }, 6000)
  }

  const handleStop = () => {
    setStatus('idle')
    addLog('system', 'Agent stopped by user')
  }

  const handleReset = () => {
    setStatus('idle')
    setGoal('')
    setMessages([])
    setLogs([])
    setPlan([])
    setLoops(0)
    setTokens(0)
    setCost(0)
    setDuration('0s')
  }

  const handleSendMessage = () => {
    if (!inputValue.trim() || status === 'idle') return
    addMessage('user', inputValue)
    setInputValue('')
    // TODO: Send to backend
  }

  // Status helpers
  const statusLabel = {
    idle: 'Ready',
    planning: 'Planning',
    executing: 'Executing',
    reflecting: 'Reflecting',
    replanning: 'Replanning',
    waiting_human: 'Waiting for input',
    done: 'Completed',
    error: 'Error'
  }

  const statusColor = {
    idle: 'var(--fg-faint)',
    planning: 'var(--warn)',
    executing: 'var(--ok)',
    reflecting: 'var(--accent)',
    replanning: 'var(--warn)',
    waiting_human: 'var(--warn)',
    done: 'var(--ok)',
    error: 'var(--err)'
  }

  const modeLabel = {
    ask: 'Ask',
    auto: 'Auto',
    yolo: 'YOLO'
  }

  return (
    <div className={`app ${darkMode ? 'dark' : 'light'}`}>
      {/* Sidebar */}
      <aside className={`sidebar ${sidebarCollapsed ? 'sidebar--collapsed' : ''}`}>
        <div className="sidebar__header">
          <div className="sidebar__logo">
            <div className="sidebar__logo-icon">Z</div>
            {!sidebarCollapsed && <span className="sidebar__logo-text">Zhulong</span>}
          </div>
          <button 
            className="sidebar__toggle"
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            title={sidebarCollapsed ? 'Expand' : 'Collapse'}
          >
            {sidebarCollapsed ? '→' : '←'}
          </button>
        </div>

        <nav className="sidebar__nav">
          <button 
            className={`sidebar__nav-item ${sidebarView === 'chat' ? 'active' : ''}`}
            onClick={() => setSidebarView('chat')}
          >
            <span className="sidebar__nav-icon">💬</span>
            {!sidebarCollapsed && <span className="sidebar__nav-label">Chat</span>}
          </button>
          <button 
            className={`sidebar__nav-item ${sidebarView === 'history' ? 'active' : ''}`}
            onClick={() => setSidebarView('history')}
          >
            <span className="sidebar__nav-icon">📋</span>
            {!sidebarCollapsed && <span className="sidebar__nav-label">History</span>}
          </button>
          <button 
            className={`sidebar__nav-item ${sidebarView === 'agents' ? 'active' : ''}`}
            onClick={() => setSidebarView('agents')}
          >
            <span className="sidebar__nav-icon">🤖</span>
            {!sidebarCollapsed && <span className="sidebar__nav-label">Agents</span>}
          </button>
        </nav>

        <div className="sidebar__footer">
          <button 
            className="sidebar__icon-btn"
            onClick={() => setDarkMode(!darkMode)}
            title={darkMode ? 'Light mode' : 'Dark mode'}
          >
            {darkMode ? '☀️' : '🌙'}
          </button>
          <button 
            className="sidebar__icon-btn"
            title="Settings"
          >
            ⚙️
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="main">
        {/* Header */}
        <header className="header">
          <div className="header__left">
            <h1 className="header__title">Autonomous Loop Agent</h1>
            <span className="header__version">v0.1.0</span>
          </div>
          <div className="header__right">
            <div className="header__mode-switcher">
              {(['ask', 'auto', 'yolo'] as ExecutionMode[]).map(mode => (
                <button
                  key={mode}
                  className={`header__mode-btn ${executionMode === mode ? 'active' : ''}`}
                  onClick={() => setExecutionMode(mode)}
                  title={`Execution mode: ${mode}`}
                >
                  {modeLabel[mode]}
                </button>
              ))}
            </div>
            <div className="header__status" style={{ color: statusColor[status] }}>
              <span className="header__status-dot" style={{ background: statusColor[status] }}></span>
              <span>{statusLabel[status]}</span>
            </div>
          </div>
        </header>

        {/* Goal Input */}
        <div className="goal-section">
          <div className="goal-input-wrapper">
            <input
              type="text"
              value={goal}
              onChange={(e) => setGoal(e.target.value)}
              placeholder="What do you want to achieve?"
              className="goal-input"
              disabled={status !== 'idle' && status !== 'done' && status !== 'error'}
              onKeyDown={(e) => e.key === 'Enter' && handleStart()}
            />
            <div className="goal-actions">
              {status === 'idle' || status === 'done' || status === 'error' ? (
                <button onClick={handleStart} className="btn btn-primary" disabled={!goal.trim()}>
                  Start
                </button>
              ) : (
                <button onClick={handleStop} className="btn btn-danger">
                  Stop
                </button>
              )}
              <button onClick={handleReset} className="btn btn-ghost">
                Reset
              </button>
            </div>
          </div>
        </div>

        {/* Transcript */}
        <div className="transcript" ref={transcriptRef}>
          {messages.length === 0 ? (
            <div className="transcript__empty">
              <div className="transcript__empty-icon">Z</div>
              <h2 className="transcript__empty-title">Zhulong</h2>
              <p className="transcript__empty-desc">Enter a goal to start the autonomous loop agent.</p>
              <div className="transcript__examples">
                <button className="transcript__example-chip" onClick={() => setGoal('Analyze the codebase and suggest improvements')}>
                  📊 Analyze codebase
                </button>
                <button className="transcript__example-chip" onClick={() => setGoal('Find and fix all TypeScript errors')}>
                  🔧 Fix TypeScript errors
                </button>
                <button className="transcript__example-chip" onClick={() => setGoal('Write unit tests for the main module')}>
                  🧪 Write unit tests
                </button>
                <button className="transcript__example-chip" onClick={() => setGoal('Search the web for latest AI news')}>
                  🌐 Search web
                </button>
              </div>
            </div>
          ) : (
            messages.map(msg => (
              <div key={msg.id} className={`message message--${msg.role}`}>
                <div className="message__avatar">
                  {msg.role === 'user' ? 'U' : msg.role === 'tool' ? '🔧' : 'Z'}
                </div>
                <div className="message__content">
                  <div className="message__header">
                    <span className="message__role">
                      {msg.role === 'user' ? 'You' : msg.role === 'tool' ? msg.toolName : 'Zhulong'}
                    </span>
                    <span className="message__time">{msg.timestamp.toLocaleTimeString()}</span>
                  </div>
                  <div className="message__text">
                    {msg.role === 'tool' ? (
                      <div className="message__tool">
                        <span className="message__tool-name">{msg.toolName}</span>
                        <span className="message__tool-status">completed</span>
                      </div>
                    ) : null}
                    {msg.content}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Input Bar */}
        {status !== 'idle' && (
          <div className="input-bar">
            <textarea
              ref={inputRef}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              placeholder={status === 'waiting_human' ? 'Provide input or approve...' : 'Add context or instructions...'}
              className="input-bar__textarea"
              rows={1}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSendMessage()
                }
              }}
            />
            <button 
              className="input-bar__send"
              onClick={handleSendMessage}
              disabled={!inputValue.trim()}
            >
              →
            </button>
          </div>
        )}
      </main>

      {/* Right Panel */}
      <aside className="right-panel">
        <div className="right-panel__tabs">
          <button 
            className={`right-panel__tab ${rightPanel === 'logs' ? 'active' : ''}`}
            onClick={() => setRightPanel('logs')}
          >
            Logs
          </button>
          <button 
            className={`right-panel__tab ${rightPanel === 'plan' ? 'active' : ''}`}
            onClick={() => setRightPanel('plan')}
          >
            Plan
          </button>
          <button 
            className={`right-panel__tab ${rightPanel === 'context' ? 'active' : ''}`}
            onClick={() => setRightPanel('context')}
          >
            Context
          </button>
          <button 
            className={`right-panel__tab ${rightPanel === 'settings' ? 'active' : ''}`}
            onClick={() => setRightPanel('settings')}
          >
            Settings
          </button>
        </div>

        <div className="right-panel__content">
          {/* Logs View */}
          {rightPanel === 'logs' && (
            <div className="logs-view">
              {logs.length === 0 ? (
                <div className="panel-empty">No logs yet</div>
              ) : (
                logs.map(log => (
                  <div key={log.id} className="log-entry">
                    <span className="log-entry__time">{log.time}</span>
                    <span className={`log-entry__phase log-entry__phase--${log.phase}`}>{log.phase}</span>
                    <span className="log-entry__event">{log.event}</span>
                    {log.detail && <span className="log-entry__detail">{log.detail}</span>}
                  </div>
                ))
              )}
            </div>
          )}

          {/* Plan View */}
          {rightPanel === 'plan' && (
            <div className="plan-view">
              {plan.length === 0 ? (
                <div className="panel-empty">
                  {status === 'idle' ? 'Start an agent to see the plan' : 'Plan will appear here'}
                </div>
              ) : (
                plan.map(step => (
                  <div key={step.id} className={`plan-step plan-step--${step.status}`}>
                    <span className="plan-step__icon">
                      {step.status === 'completed' ? '✓' : step.status === 'running' ? '◉' : '○'}
                    </span>
                    <span className="plan-step__desc">{step.description}</span>
                  </div>
                ))
              )}
            </div>
          )}

          {/* Context View */}
          {rightPanel === 'context' && (
            <div className="context-view">
              <div className="context-section">
                <h3 className="context-section__title">Token Usage</h3>
                <div className="context-stats">
                  <div className="context-stat">
                    <span className="context-stat__label">Total</span>
                    <span className="context-stat__value">{tokens.toLocaleString()}</span>
                  </div>
                  <div className="context-stat">
                    <span className="context-stat__label">Cost</span>
                    <span className="context-stat__value">¥{cost.toFixed(2)}</span>
                  </div>
                </div>
              </div>
              <div className="context-section">
                <h3 className="context-section__title">Session</h3>
                <div className="context-stats">
                  <div className="context-stat">
                    <span className="context-stat__label">Loops</span>
                    <span className="context-stat__value">{loops}</span>
                  </div>
                  <div className="context-stat">
                    <span className="context-stat__label">Duration</span>
                    <span className="context-stat__value">{duration}</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Settings View */}
          {rightPanel === 'settings' && (
            <div className="settings-view">
              <div className="settings-section">
                <h3 className="settings-section__title">Execution Mode</h3>
                <div className="settings-options">
                  {(['ask', 'auto', 'yolo'] as ExecutionMode[]).map(mode => (
                    <label key={mode} className="settings-option">
                      <input
                        type="radio"
                        name="mode"
                        value={mode}
                        checked={executionMode === mode}
                        onChange={() => setExecutionMode(mode)}
                      />
                      <span className="settings-option__label">{modeLabel[mode]}</span>
                      <span className="settings-option__desc">
                        {mode === 'ask' ? 'Ask for approval on risky operations' :
                         mode === 'auto' ? 'Auto-approve low risk, ask for high risk' :
                         'Trust everything, no interruptions'}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="settings-section">
                <h3 className="settings-section__title">Appearance</h3>
                <label className="settings-option">
                  <input
                    type="checkbox"
                    checked={darkMode}
                    onChange={() => setDarkMode(!darkMode)}
                  />
                  <span className="settings-option__label">Dark mode</span>
                </label>
              </div>
            </div>
          )}
        </div>
      </aside>
    </div>
  )
}

export default App
