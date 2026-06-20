import { useState } from 'react'
import { Greet, GetVersion } from '../wailsjs/go/main/App'

function App() {
  const [result, setResult] = useState<string>('')
  const [name, setName] = useState<string>('')
  const [version, setVersion] = useState<string>('')

  const handleGreet = async () => {
    const greeting = await Greet(name)
    setResult(greeting)
  }

  const handleGetVersion = async () => {
    const v = await GetVersion()
    setVersion(v)
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="logo">
          <h1>Zhulong</h1>
          <p className="subtitle">Autonomous Loop Agent</p>
        </div>
      </header>

      <main className="app-main">
        <div className="card">
          <h2>Welcome</h2>
          <p>Zhulong is an autonomous loop agent powered by DeepSeek.</p>
          
          <div className="input-group">
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Enter your name"
              className="input"
            />
            <button onClick={handleGreet} className="button button-primary">
              Greet
            </button>
          </div>

          {result && (
            <div className="result">
              <p>{result}</p>
            </div>
          )}
        </div>

        <div className="card">
          <h2>Version</h2>
          <button onClick={handleGetVersion} className="button button-secondary">
            Get Version
          </button>
          {version && (
            <div className="result">
              <p>Version: {version}</p>
            </div>
          )}
        </div>
      </main>

      <footer className="app-footer">
        <p>Zhulong v0.1.0 - Built with Wails + React</p>
      </footer>
    </div>
  )
}

export default App
