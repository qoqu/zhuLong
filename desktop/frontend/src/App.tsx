import { useState, useRef, useEffect } from 'react'

interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date
}

interface LogEntry {
  id: string
  time: string
  phase: string
  event: string
  detail?: string
}

function App() {
  const [goal, setGoal] = useState<string>('')
  const [status, setStatus] = useState<'idle' | 'planning' | 'executing' | 'reflecting' | 'done' | 'error'>('idle')
  const [messages, setMessages] = useState<Message[]>([])
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [rightPanel, setRightPanel] = useState<'logs' | 'stats' | 'plan'>('logs')
  const [darkMode, setDarkMode] = useState(true)
  const transcriptRef = useRef<HTMLDivElement>(null)

  // Stats
  const [loops, setLoops] = useState(0)
  const [tokens, setTokens] = useState(0)
  const [cost, setCost] = useState(0)
  const [duration, setDuration] = useState('0s')

  useEffect(() => {
    if (transcriptRef.current) {
      transcriptRef.current.scrollTop = transcriptRef.current.scrollHeight
    }
  }, [messages])

  const addLog = (phase: string, event: string, detail?: string) => {
    setLogs(prev => [...prev, {
      id: Date.now().toString(),
      time: new Date().toLocaleTimeString(),
      phase,
      event,
      detail
    }])
  }

  const handleStart = () => {
    if (!goal.trim()) return
    
    setStatus('planning')
    setMessages([{
      id: Date.now().toString(),
      role: 'user',
      content: goal,
      timestamp: new Date()
    }])
    addLog('system', 'Agent started', `Goal: ${goal}`)

    // Simulate agent loop
    setTimeout(() => {
      setStatus('executing')
      addLog('planning', 'Plan created', '3 steps')
      setMessages(prev => [...prev, {
        id: Date.now().toString(),
        role: 'assistant',
        content: 'I\'ll help you achieve this goal. Let me start by analyzing the task...',
        timestamp: new Date()
      }])
    }, 1500)

    setTimeout(() => {
      setStatus('reflecting')
      addLog('executing', 'Step 1/3 completed', 'read_file: main.go')
      setLoops(1)
      setTokens(1234)
      setCost(0.02)
    }, 3000)

    setTimeout(() => {
      setStatus('done')
      addLog('reflecting', 'Goal achieved')
      setMessages(prev => [...prev, {
        id: Date.now().toString(),
        role: 'assistant',
        content: 'Task completed successfully! Here\'s what I found...',
        timestamp: new Date()
      }])
      setDuration('4.5s')
    }, 4500)
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
    setLoops(0)
    setTokens(0)
    setCost(0)
    setDuration('0s')
  }

  const statusLabel = {
    idle: 'Ready',
    planning: 'Planning',
    executing: 'Executing',
    reflecting: 'Reflecting',
    done: 'Completed',
    error: 'Error'
  }

  return (
    <div className={`app ${darkMode ? 'dark' : 'light'}`}>
      {/* Sidebar */}
      <aside className={`sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div className="sidebar-header">
          <div className="logo">
            <span className="logo-icon">Z</span>
            {!sidebarCollapsed && <span className="logo-text">Zhulong</span>}
          </div>
          <button 
            className="btn-icon"
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
          >
            {sidebarCollapsed ? '→' : '←'}
          </button>
        </div>

        <nav className="sidebar-nav">
          <button className="nav-item active">
            <span className="nav-icon">💬</span>
            {!sidebarCollapsed && <span className="nav-label">Chat</span>}
          </button>
          <button className="nav-item">
            <span className="nav-icon">📋</span>
            {!sidebarCollapsed && <span className="nav-label">History</span>}
          </button>
          <button className="nav-item">
            <span className="nav-icon">⚙️</span>
            {!sidebarCollapsed && <span className="nav-label">Settings</span>}
          </button>
        </nav>

        <div className="sidebar-footer">
          <button 
            className="btn-icon"
            onClick={() => setDarkMode(!darkMode)}
          >
            {darkMode ? '☀️' : '🌙'}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="main-content">
        {/* Header */}
        <header className="header">
          <div className="header-left">
            <h1>Autonomous Loop Agent</h1>
            <span className="version">v0.1.0</span>
          </div>
          <div className="header-right">
            <div className={`status-badge ${status}`}>
              <span className="status-dot"></span>
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
            <div className="transcript-empty">
              <div className="empty-icon">Z</div>
              <h2>Zhulong</h2>
              <p>Enter a goal to start the autonomous loop agent.</p>
              <div className="examples">
                <button className="example-chip" onClick={() => setGoal('Analyze the codebase and suggest improvements')}>
                  Analyze codebase
                </button>
                <button className="example-chip" onClick={() => setGoal('Find and fix all TypeScript errors')}>
                  Fix TypeScript errors
                </button>
                <button className="example-chip" onClick={() => setGoal('Write unit tests for the main module')}>
                  Write unit tests
                </button>
              </div>
            </div>
          ) : (
            messages.map(msg => (
              <div key={msg.id} className={`message ${msg.role}`}>
                <div className="message-avatar">
                  {msg.role === 'user' ? 'U' : 'Z'}
                </div>
                <div className="message-content">
                  <div className="message-header">
                    <span className="message-role">{msg.role === 'user' ? 'You' : 'Zhulong'}</span>
                    <span className="message-time">{msg.timestamp.toLocaleTimeString()}</span>
                  </div>
                  <div className="message-text">{msg.content}</div>
                </div>
              </div>
            ))
          )}
        </div>
      </main>

      {/* Right Panel */}
      <aside className="right-panel">
        <div className="panel-tabs">
          <button 
            className={`panel-tab ${rightPanel === 'logs' ? 'active' : ''}`}
            onClick={() => setRightPanel('logs')}
          >
            Logs
          </button>
          <button 
            className={`panel-tab ${rightPanel === 'stats' ? 'active' : ''}`}
            onClick={() => setRightPanel('stats')}
          >
            Stats
          </button>
          <button 
            className={`panel-tab ${rightPanel === 'plan' ? 'active' : ''}`}
            onClick={() => setRightPanel('plan')}
          >
            Plan
          </button>
        </div>

        <div className="panel-content">
          {rightPanel === 'logs' && (
            <div className="logs-view">
              {logs.length === 0 ? (
                <div className="panel-empty">No logs yet</div>
              ) : (
                logs.map(log => (
                  <div key={log.id} className="log-entry">
                    <span className="log-time">{log.time}</span>
                    <span className={`log-phase ${log.phase}`}>{log.phase}</span>
                    <span className="log-event">{log.event}</span>
                    {log.detail && <span className="log-detail">{log.detail}</span>}
                  </div>
                ))
              )}
            </div>
          )}

          {rightPanel === 'stats' && (
            <div className="stats-view">
              <div className="stat-card">
                <div className="stat-label">Loops</div>
                <div className="stat-value">{loops}</div>
              </div>
              <div className="stat-card">
                <div className="stat-label">Tokens</div>
                <div className="stat-value">{tokens.toLocaleString()}</div>
              </div>
              <div className="stat-card">
                <div className="stat-label">Cost</div>
                <div className="stat-value">¥{cost.toFixed(2)}</div>
              </div>
              <div className="stat-card">
                <div className="stat-label">Duration</div>
                <div className="stat-value">{duration}</div>
              </div>
            </div>
          )}

          {rightPanel === 'plan' && (
            <div className="plan-view">
              <div className="panel-empty">
                {status === 'idle' ? 'Start an agent to see the plan' : 'Plan will appear here'}
              </div>
            </div>
          )}
        </div>
      </aside>
    </div>
  )
}

export default App
