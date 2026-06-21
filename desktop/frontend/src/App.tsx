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
type Language = 'en' | 'zh'

// Translations
const translations = {
  en: {
    // Header
    title: 'Autonomous Loop Agent',
    version: 'v0.1.0',
    
    // Sidebar
    chat: 'Chat',
    history: 'History',
    agents: 'Agents',
    
    // Status
    ready: 'Ready',
    planning: 'Planning',
    executing: 'Executing',
    reflecting: 'Reflecting',
    replanning: 'Replanning',
    waiting_human: 'Waiting for input',
    done: 'Completed',
    error: 'Error',
    
    // Modes
    ask: 'Ask',
    auto: 'Auto',
    yolo: 'YOLO',
    
    // Goal
    goal_placeholder: 'What do you want to achieve?',
    start: 'Start',
    stop: 'Stop',
    reset: 'Reset',
    
    // Examples
    analyze_codebase: 'Analyze codebase',
    fix_typescript: 'Fix TypeScript errors',
    write_tests: 'Write unit tests',
    search_web: 'Search web',
    
    // Messages
    you: 'You',
    zhulong: 'Zhulong',
    completed: 'completed',
    
    // Input
    input_placeholder: 'Add context or instructions...',
    input_placeholder_waiting: 'Provide input or approve...',
    
    // Right Panel
    logs: 'Logs',
    plan: 'Plan',
    context: 'Context',
    settings: 'Settings',
    
    // Empty states
    no_logs: 'No logs yet',
    no_plan_idle: 'Start an agent to see the plan',
    no_plan_running: 'Plan will appear here',
    
    // Context
    token_usage: 'Token Usage',
    total: 'Total',
    cost: 'Cost',
    session: 'Session',
    loops: 'Loops',
    duration: 'Duration',
    
    // Settings
    execution_mode: 'Execution Mode',
    ask_desc: 'Ask for approval on risky operations',
    auto_desc: 'Auto-approve low risk, ask for high risk',
    yolo_desc: 'Trust everything, no interruptions',
    appearance: 'Appearance',
    dark_mode: 'Dark mode',
    language: 'Language',
    
    // Agent messages
    agent_started: 'Agent started',
    execution_mode_label: 'Execution mode',
    plan_created: 'Plan created',
    steps: 'steps',
    goal_achieved: 'Goal achieved',
    agent_stopped: 'Agent stopped by user',
    analyzing: "I'll help you achieve this goal. Let me start by analyzing the task...",
    completed_message: "Task completed successfully! Here's what I found:\n\n1. The codebase compiles without errors\n2. All tests pass\n3. No security issues detected",
  },
  zh: {
    // Header
    title: '自主循环 Agent',
    version: 'v0.1.0',
    
    // Sidebar
    chat: '对话',
    history: '历史',
    agents: '智能体',
    
    // Status
    ready: '就绪',
    planning: '规划中',
    executing: '执行中',
    reflecting: '反省中',
    replanning: '重规划中',
    waiting_human: '等待输入',
    done: '已完成',
    error: '错误',
    
    // Modes
    ask: '询问',
    auto: '自动',
    yolo: 'YOLO',
    
    // Goal
    goal_placeholder: '你想要实现什么目标？',
    start: '开始',
    stop: '停止',
    reset: '重置',
    
    // Examples
    analyze_codebase: '分析代码库',
    fix_typescript: '修复 TypeScript 错误',
    write_tests: '编写单元测试',
    search_web: '搜索网络',
    
    // Messages
    you: '你',
    zhulong: '烛龙',
    completed: '已完成',
    
    // Input
    input_placeholder: '添加上下文或指令...',
    input_placeholder_waiting: '提供输入或确认...',
    
    // Right Panel
    logs: '日志',
    plan: '计划',
    context: '上下文',
    settings: '设置',
    
    // Empty states
    no_logs: '暂无日志',
    no_plan_idle: '启动 Agent 查看计划',
    no_plan_running: '计划将在此显示',
    
    // Context
    token_usage: 'Token 使用',
    total: '总计',
    cost: '费用',
    session: '会话',
    loops: '循环',
    duration: '耗时',
    
    // Settings
    execution_mode: '执行模式',
    ask_desc: '每次风险操作都询问用户',
    auto_desc: '低风险自动执行，高风险才询问',
    yolo_desc: '全部自动执行，不询问',
    appearance: '外观',
    dark_mode: '深色模式',
    language: '语言',
    
    // Agent messages
    agent_started: 'Agent 已启动',
    execution_mode_label: '执行模式',
    plan_created: '计划已创建',
    steps: '步骤',
    goal_achieved: '目标已达成',
    agent_stopped: '用户停止了 Agent',
    analyzing: '我来帮你实现这个目标。让我先分析一下任务...',
    completed_message: '任务完成！以下是我的发现：\n\n1. 代码库编译无错误\n2. 所有测试通过\n3. 未发现安全问题',
  }
}

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
  const [language, setLanguage] = useState<Language>('zh') // Default to Chinese
  
  // Translation helper
  const t = translations[language]
  
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
    addLog('system', t.agent_started, `Goal: ${goal}`)
    addLog('system', `${t.execution_mode_label}: ${executionMode}`)

    // Simulate agent loop
    setTimeout(() => {
      setStatus('executing')
      addLog('planning', t.plan_created, `3 ${t.steps}`)
      setPlan([
        { id: '1', description: language === 'zh' ? '分析任务并收集信息' : 'Analyze the task and gather information', status: 'completed' },
        { id: '2', description: language === 'zh' ? '执行主要操作' : 'Execute the main action', status: 'running' },
        { id: '3', description: language === 'zh' ? '验证并总结结果' : 'Verify and summarize results', status: 'pending' }
      ])
      addMessage('assistant', t.analyzing)
    }, 1500)

    setTimeout(() => {
      addLog('executing', language === 'zh' ? '步骤 1/3 完成' : 'Step 1/3 completed', 'read_file: main.go')
      addMessage('tool', language === 'zh' ? '读取文件: main.go' : 'Reading file: main.go', 'read_file')
      setLoops(1)
      setTokens(1234)
      setCost(0.02)
    }, 3000)

    setTimeout(() => {
      setStatus('reflecting')
      addLog('executing', language === 'zh' ? '步骤 2/3 完成' : 'Step 2/3 completed', 'execute_command: go build')
      addMessage('tool', language === 'zh' ? '执行: go build ./...' : 'Executing: go build ./...', 'execute_command')
      setPlan(prev => prev.map(s => s.id === '2' ? { ...s, status: 'completed' } : s))
      setTokens(2500)
      setCost(0.04)
    }, 4500)

    setTimeout(() => {
      setStatus('done')
      addLog('reflecting', t.goal_achieved)
      addMessage('assistant', t.completed_message)
      setPlan(prev => prev.map(s => s.id === '3' ? { ...s, status: 'completed' } : s))
      setDuration('6.2s')
      setTokens(3800)
      setCost(0.06)
    }, 6000)
  }

  const handleStop = () => {
    setStatus('idle')
    addLog('system', t.agent_stopped)
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
    idle: t.ready,
    planning: t.planning,
    executing: t.executing,
    reflecting: t.reflecting,
    replanning: t.replanning,
    waiting_human: t.waiting_human,
    done: t.done,
    error: t.error
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
    ask: t.ask,
    auto: t.auto,
    yolo: t.yolo
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
            title={sidebarCollapsed ? (language === 'zh' ? '展开' : 'Expand') : (language === 'zh' ? '折叠' : 'Collapse')}
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
            {!sidebarCollapsed && <span className="sidebar__nav-label">{t.chat}</span>}
          </button>
          <button 
            className={`sidebar__nav-item ${sidebarView === 'history' ? 'active' : ''}`}
            onClick={() => setSidebarView('history')}
          >
            <span className="sidebar__nav-icon">📋</span>
            {!sidebarCollapsed && <span className="sidebar__nav-label">{t.history}</span>}
          </button>
          <button 
            className={`sidebar__nav-item ${sidebarView === 'agents' ? 'active' : ''}`}
            onClick={() => setSidebarView('agents')}
          >
            <span className="sidebar__nav-icon">🤖</span>
            {!sidebarCollapsed && <span className="sidebar__nav-label">{t.agents}</span>}
          </button>
        </nav>

        <div className="sidebar__footer">
          <button 
            className="sidebar__icon-btn"
            onClick={() => setDarkMode(!darkMode)}
            title={darkMode ? (language === 'zh' ? '浅色模式' : 'Light mode') : (language === 'zh' ? '深色模式' : 'Dark mode')}
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
            <h1 className="header__title">{t.title}</h1>
            <span className="header__version">{t.version}</span>
          </div>
          <div className="header__right">
            <div className="header__mode-switcher">
              {(['ask', 'auto', 'yolo'] as ExecutionMode[]).map(mode => (
                <button
                  key={mode}
                  className={`header__mode-btn ${executionMode === mode ? 'active' : ''}`}
                  onClick={() => setExecutionMode(mode)}
                  title={`${t.execution_mode}: ${mode}`}
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
              placeholder={t.goal_placeholder}
              className="goal-input"
              disabled={status !== 'idle' && status !== 'done' && status !== 'error'}
              onKeyDown={(e) => e.key === 'Enter' && handleStart()}
            />
            <div className="goal-actions">
              {status === 'idle' || status === 'done' || status === 'error' ? (
                <button onClick={handleStart} className="btn btn-primary" disabled={!goal.trim()}>
                  {t.start}
                </button>
              ) : (
                <button onClick={handleStop} className="btn btn-danger">
                  {t.stop}
                </button>
              )}
              <button onClick={handleReset} className="btn btn-ghost">
                {t.reset}
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
              <p className="transcript__empty-desc">{language === 'zh' ? '输入目标开始自主循环 Agent' : 'Enter a goal to start the autonomous loop agent.'}</p>
              <div className="transcript__examples">
                <button className="transcript__example-chip" onClick={() => setGoal(language === 'zh' ? '分析代码库并提出改进建议' : 'Analyze the codebase and suggest improvements')}>
                  📊 {t.analyze_codebase}
                </button>
                <button className="transcript__example-chip" onClick={() => setGoal(language === 'zh' ? '查找并修复所有 TypeScript 错误' : 'Find and fix all TypeScript errors')}>
                  🔧 {t.fix_typescript}
                </button>
                <button className="transcript__example-chip" onClick={() => setGoal(language === 'zh' ? '为主模块编写单元测试' : 'Write unit tests for the main module')}>
                  🧪 {t.write_tests}
                </button>
                <button className="transcript__example-chip" onClick={() => setGoal(language === 'zh' ? '搜索最新 AI 新闻' : 'Search the web for latest AI news')}>
                  🌐 {t.search_web}
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
                      {msg.role === 'user' ? t.you : msg.role === 'tool' ? msg.toolName : t.zhulong}
                    </span>
                    <span className="message__time">{msg.timestamp.toLocaleTimeString()}</span>
                  </div>
                  <div className="message__text">
                    {msg.role === 'tool' ? (
                      <div className="message__tool">
                        <span className="message__tool-name">{msg.toolName}</span>
                        <span className="message__tool-status">{t.completed}</span>
                      </div>
                    ) : null}
                    {msg.content}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Input Bar - Bottom Center */}
        {status !== 'idle' && (
          <div className="input-bar">
            <div>
              <textarea
                ref={inputRef}
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                placeholder={status === 'waiting_human' ? t.input_placeholder_waiting : t.input_placeholder}
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
            {t.logs}
          </button>
          <button 
            className={`right-panel__tab ${rightPanel === 'plan' ? 'active' : ''}`}
            onClick={() => setRightPanel('plan')}
          >
            {t.plan}
          </button>
          <button 
            className={`right-panel__tab ${rightPanel === 'context' ? 'active' : ''}`}
            onClick={() => setRightPanel('context')}
          >
            {t.context}
          </button>
          <button 
            className={`right-panel__tab ${rightPanel === 'settings' ? 'active' : ''}`}
            onClick={() => setRightPanel('settings')}
          >
            {t.settings}
          </button>
        </div>

        <div className="right-panel__content">
          {/* Logs View */}
          {rightPanel === 'logs' && (
            <div className="logs-view">
              {logs.length === 0 ? (
                <div className="panel-empty">{t.no_logs}</div>
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
                  {status === 'idle' ? t.no_plan_idle : t.no_plan_running}
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
                <h3 className="context-section__title">{t.token_usage}</h3>
                <div className="context-stats">
                  <div className="context-stat">
                    <span className="context-stat__label">{t.total}</span>
                    <span className="context-stat__value">{tokens.toLocaleString()}</span>
                  </div>
                  <div className="context-stat">
                    <span className="context-stat__label">{t.cost}</span>
                    <span className="context-stat__value">¥{cost.toFixed(2)}</span>
                  </div>
                </div>
              </div>
              <div className="context-section">
                <h3 className="context-section__title">{t.session}</h3>
                <div className="context-stats">
                  <div className="context-stat">
                    <span className="context-stat__label">{t.loops}</span>
                    <span className="context-stat__value">{loops}</span>
                  </div>
                  <div className="context-stat">
                    <span className="context-stat__label">{t.duration}</span>
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
                <h3 className="settings-section__title">{t.execution_mode}</h3>
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
                        {mode === 'ask' ? t.ask_desc :
                         mode === 'auto' ? t.auto_desc :
                         t.yolo_desc}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="settings-section">
                <h3 className="settings-section__title">{t.appearance}</h3>
                <label className="settings-option">
                  <input
                    type="checkbox"
                    checked={darkMode}
                    onChange={() => setDarkMode(!darkMode)}
                  />
                  <span className="settings-option__label">{t.dark_mode}</span>
                </label>
              </div>
              <div className="settings-section">
                <h3 className="settings-section__title">{t.language}</h3>
                <div className="settings-options">
                  <label className="settings-option">
                    <input
                      type="radio"
                      name="language"
                      value="zh"
                      checked={language === 'zh'}
                      onChange={() => setLanguage('zh')}
                    />
                    <span className="settings-option__label">中文</span>
                  </label>
                  <label className="settings-option">
                    <input
                      type="radio"
                      name="language"
                      value="en"
                      checked={language === 'en'}
                      onChange={() => setLanguage('en')}
                    />
                    <span className="settings-option__label">English</span>
                  </label>
                </div>
              </div>
            </div>
          )}
        </div>
      </aside>
    </div>
  )
}

export default App
