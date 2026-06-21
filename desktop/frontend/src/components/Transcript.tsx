import { useEffect, useRef } from 'react'
import type { Message, Language, LogEntry, PlanStep, AgentStatus } from '../types'

interface TranscriptProps {
  language: Language
  messages: Message[]
  logs: LogEntry[]
  plan: PlanStep[]
  status: AgentStatus
}

export function Transcript(props: TranscriptProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) {
      ref.current.scrollTop = ref.current.scrollHeight
    }
  }, [props.messages, props.logs])

  return (
    <div className="transcript" ref={ref}>
      {/* Inline plan summary at top */}
      {props.plan.length > 0 && (
        <div className="plan-card">
          <div className="plan-card__title">
            {props.language === 'zh' ? '计划' : 'Plan'}
            <span className="plan-card__status" data-status={props.status}>
              {props.status}
            </span>
          </div>
          {props.plan.map((p) => (
            <div key={p.id} className={`plan-card__step plan-card__step--${p.status}`}>
              <span className="plan-card__icon">
                {p.status === 'completed' ? '✓' : p.status === 'running' ? '◉' : p.status === 'failed' ? '✕' : '○'}
              </span>
              <span className="plan-card__desc">{p.description}</span>
            </div>
          ))}
        </div>
      )}

      {/* Recent logs */}
      {props.logs.length > 0 && (
        <details className="log-card" open>
          <summary className="log-card__title">
            {props.language === 'zh' ? '日志' : 'Logs'} ({props.logs.length})
          </summary>
          <div className="log-card__body">
            {props.logs.slice(-12).map((l) => (
              <div key={l.id} className="log-card__entry">
                <span className="log-card__time">{l.time}</span>
                <span className={`log-card__phase log-card__phase--${l.phase}`}>{l.phase}</span>
                <span className="log-card__event">{l.event}</span>
                {l.detail && <span className="log-card__detail">{l.detail}</span>}
              </div>
            ))}
          </div>
        </details>
      )}

      {/* Messages */}
      {props.messages.map((m) => (
        <MessageRow key={m.id} message={m} />
      ))}
    </div>
  )
}

function MessageRow(props: { message: Message }) {
  const m = props.message
  if (m.role === 'user') {
    return (
      <div className="msg msg--user">
        <div className="msg__bubble msg__bubble--user">
          <div className="msg__text">{m.content}</div>
          {m.toolCount !== undefined && m.toolCount > 0 && (
            <div className="msg__meta">{m.toolCount} 个工具</div>
          )}
        </div>
        <div className="msg__avatar msg__avatar--user">U</div>
      </div>
    )
  }
  if (m.role === 'tool') {
    return (
      <div className="msg msg--tool">
        <div className="msg__icon">🔧</div>
        <div className="msg__bubble msg__bubble--tool">
          <strong>{m.toolName}</strong>
          <span>{m.content}</span>
        </div>
      </div>
    )
  }
  return (
    <div className="msg msg--assistant">
      <div className="msg__avatar msg__avatar--assistant">Z</div>
      <div className="msg__bubble msg__bubble--assistant">
        <div className="msg__text">{m.content}</div>
      </div>
    </div>
  )
}
