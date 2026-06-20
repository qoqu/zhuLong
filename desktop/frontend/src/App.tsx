import { useState } from 'react'

function App() {
  const [goal, setGoal] = useState<string>('')
  const [status, setStatus] = useState<string>('idle')
  const [logs, setLogs] = useState<string[]>([])

  const handleStart = () => {
    if (!goal.trim()) return
    setStatus('running')
    setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] Starting agent with goal: ${goal}`])
    
    // TODO: Connect to Go backend
    setTimeout(() => {
      setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] Agent completed`])
      setStatus('completed')
    }, 2000)
  }

  const handleStop = () => {
    setStatus('idle')
    setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] Agent stopped`])
  }

  const handleClear = () => {
    setLogs([])
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="logo">
          <h1>Zhulong</h1>
          <p className="subtitle">Autonomous Loop Agent</p>
        </div>
        <div className="version">v0.1.0</div>
      </header>

      <main className="app-main">
        <div className="card goal-card">
          <h2>Goal</h2>
          <p>Enter your goal and the agent will work to achieve it.</p>
          
          <div className="input-group">
            <input
              type="text"
              value={goal}
              onChange={(e) => setGoal(e.target.value)}
              placeholder="What do you want to achieve?"
              className="input"
              disabled={status === 'running'}
            />
            {status === 'running' ? (
              <button onClick={handleStop} className="button button-danger">
                Stop
              </button>
            ) : (
              <button onClick={handleStart} className="button button-primary" disabled={!goal.trim()}>
                Start
              </button>
            )}
          </div>

          <div className="status-bar">
            <div className={`status-indicator ${status}`}>
              <span className="status-dot"></span>
              <span className="status-text">{status}</span>
            </div>
          </div>
        </div>

        <div className="card logs-card">
          <div className="logs-header">
            <h2>Logs</h2>
            <button onClick={handleClear} className="button button-secondary button-sm">
              Clear
            </button>
          </div>
          
          <div className="logs-container">
            {logs.length === 0 ? (
              <div className="logs-empty">
                <p>No logs yet. Start the agent to see activity.</p>
              </div>
            ) : (
              logs.map((log, index) => (
                <div key={index} className="log-entry">
                  {log}
                </div>
              ))
            )}
          </div>
        </div>

        <div className="card stats-card">
          <h2>Statistics</h2>
          <div className="stats-grid">
            <div className="stat">
              <span className="stat-label">Loops</span>
              <span className="stat-value">0</span>
            </div>
            <div className="stat">
              <span className="stat-label">Tokens</span>
              <span className="stat-value">0</span>
            </div>
            <div className="stat">
              <span className="stat-label">Cost</span>
              <span className="stat-value">¥0.00</span>
            </div>
            <div className="stat">
              <span className="stat-label">Duration</span>
              <span className="stat-value">0s</span>
            </div>
          </div>
        </div>
      </main>

      <footer className="app-footer">
        <p>Zhulong v0.1.0 - Powered by DeepSeek</p>
      </footer>
    </div>
  )
}

export default App
