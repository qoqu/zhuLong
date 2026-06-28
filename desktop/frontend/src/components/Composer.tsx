import type { ExecutionMode, Language, AgentStatus } from '../types'
import { useT } from '../i18n'

interface ComposerProps {
  language: Language
  status: AgentStatus
  executionMode: ExecutionMode
  onChangeExecutionMode: (m: ExecutionMode) => void
  model: string
  onChangeModel: (m: string) => void
  temperature: string
  onChangeTemperature: (t: string) => void
  onSend: (text: string) => void
  onStop: () => void
  onReset: () => void
  value: string
  onChange: (v: string) => void
  waitingHuman: boolean
}

// TODO: Model list is hardcoded; ideally providers should be passed as props from the backend.
const models = [
  'deepseek-v4-flash',
  'deepseek-v4-pro',
]

const temperatures = [
  { v: 'auto', label: 'auto' },
  { v: '0.0', label: '0.0' },
  { v: '0.3', label: '0.3' },
  { v: '0.7', label: '0.7' },
  { v: '1.0', label: '1.0' },
]

export function Composer(props: ComposerProps) {
  const t = useT(props.language)
  const isRunning =
    props.status !== 'idle' && props.status !== 'done' && props.status !== 'error'
  const placeholder = props.waitingHuman ? t.inputPlaceholderWaiting : t.placeholder

  const handleSend = () => {
    if (!props.value.trim()) return
    props.onSend(props.value.trim())
  }

  return (
    <div className="composer">
      <div className="composer__row">
        <textarea
          className="composer__textarea"
          placeholder={placeholder}
          value={props.value}
          onChange={(e) => props.onChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              handleSend()
            }
          }}
          rows={1}
        />
        {isRunning ? (
          <button className="composer__send composer__send--stop" onClick={props.onStop} title={t.stop}>
            ■
          </button>
        ) : (
          <button
            className="composer__send"
            onClick={handleSend}
            disabled={!props.value.trim()}
            title={t.send}
          >
            ↑
          </button>
        )}
      </div>
      <div className="composer__toolbar">
        <button className="composer__icon" title="Slash command">/</button>

        <div className="composer__group">
          {(['ask', 'auto', 'yolo'] as ExecutionMode[]).map((m) => (
            <button
              key={m}
              className={`pill pill--mode ${props.executionMode === m ? 'active' : ''} ${
                m === 'yolo' ? 'pill--yolo' : ''
              }`}
              onClick={() => props.onChangeExecutionMode(m)}
            >
              {m === 'ask' ? t.modeAsk : m === 'auto' ? t.modeAuto : t.modeYolo}
            </button>
          ))}
        </div>

        <select
          className="composer__select"
          value={props.model}
          onChange={(e) => props.onChangeModel(e.target.value)}
        >
          {models.map((m) => (
            <option key={m} value={m}>
              {m}
            </option>
          ))}
        </select>

        <select
          className="composer__select"
          value={props.temperature}
          onChange={(e) => props.onChangeTemperature(e.target.value)}
        >
          {temperatures.map((tt) => (
            <option key={tt.v} value={tt.v}>
              {tt.label}
            </option>
          ))}
        </select>

        <div className="composer__spacer" />

        <button className="composer__icon" title="More">⋯</button>
      </div>
    </div>
  )
}
