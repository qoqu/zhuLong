import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { Transcript } from './components/Transcript'
import { Composer } from './components/Composer'
import { RightPanel } from './components/RightPanel'
import { StatusBar } from './components/StatusBar'
import { mockStats, exampleGoals } from './data/mock'
import type {
  AgentStatus,
  ExecutionMode,
  InputMode,
  Language,
  RightPanelTab,
  SidebarView,
  Message,
} from './types'
import { useT } from './i18n'

function App() {
  // Theme
  const [darkMode, setDarkMode] = useState(true)
  const [language, setLanguage] = useState<Language>('zh')

  // Sidebar
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [sidebarView, setSidebarView] = useState<SidebarView>('projects')
  const [activeAgentId, setActiveAgentId] = useState('auto')
  const [activeSessionId, setActiveSessionId] = useState('s2')

  // Right panel
  const [rightTab, setRightTab] = useState<RightPanelTab>('overview')

  // Agent
  const [status, setStatus] = useState<AgentStatus>('idle')
  const [executionMode, setExecutionMode] = useState<ExecutionMode>('auto')
  const [inputMode, setInputMode] = useState<InputMode>('normal')
  const [model, setModel] = useState('deepseek-v4-flash')
  const [temperature, setTemperature] = useState('auto')
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')

  const handleSend = (text: string) => {
    const userMsg: Message = {
      id: 'u' + Date.now(),
      role: 'user',
      content: text,
      timestamp: new Date(),
    }
    setMessages((m) => [...m, userMsg])
    setInput('')
    setStatus('executing')

    // Simulated agent response
    setTimeout(() => {
      setStatus('reflecting')
      setTimeout(() => {
        setStatus('done')
        setMessages((m) => [
          ...m,
          {
            id: 'a' + Date.now(),
            role: 'assistant',
            content: '任务已完成。我已经处理了你的请求。',
            timestamp: new Date(),
          },
        ])
      }, 1200)
    }, 1500)
  }

  const handleStop = () => {
    setStatus('idle')
  }

  const handleReset = () => {
    setMessages([])
    setStatus('idle')
  }

  const handleNewSession = () => {
    setMessages([])
    setStatus('idle')
  }

  return (
    <div className={`app ${darkMode ? 'theme--dark' : 'theme--light'}`}>
      <Sidebar
        language={language}
        onLanguageChange={setLanguage}
        collapsed={sidebarCollapsed}
        onToggleCollapsed={() => setSidebarCollapsed((v) => !v)}
        view={sidebarView}
        onChangeView={setSidebarView}
        activeAgentId={activeAgentId}
        onSelectAgent={setActiveAgentId}
        activeSessionId={activeSessionId}
        onSelectSession={setActiveSessionId}
        onNewSession={handleNewSession}
        onNewAgent={() => {}}
        onOpenCommandPalette={() => {}}
      />

      <main className="main">
        <TopBar
          language={language}
          sessionTitle="mnemonic全笔记2.mcp"
          sessionScope="Global"
          onRename={() => {}}
          onExport={() => {}}
          status={status}
        />

        {messages.length === 0 ? (
          <EmptyState
            language={language}
            onPick={(text) => setInput(text)}
          />
        ) : (
          <Transcript language={language} messages={messages} />
        )}

        <div className="main__composer">
          <Composer
            language={language}
            status={status}
            executionMode={executionMode}
            onChangeExecutionMode={setExecutionMode}
            inputMode={inputMode}
            onChangeInputMode={setInputMode}
            model={model}
            onChangeModel={setModel}
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
      </main>

      <RightPanel
        language={language}
        tab={rightTab}
        onChangeTab={setRightTab}
        stats={mockStats}
      />

      <StatusBar
        language={language}
        status={status}
        model={model}
        stats={mockStats}
        darkMode={darkMode}
        onToggleDarkMode={() => setDarkMode((v) => !v)}
        languageToggle={() => setLanguage((l) => (l === 'en' ? 'zh' : 'en'))}
      />
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
