import { useEffect, useRef } from 'react'
import type { Message, Language, LogEntry, PlanStep, AgentStatus } from '../types'
import { resolveLanguage } from '../i18n'

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
    <div className="transcript-layout">
      {/* Sticky header: plan + logs */}
      <div className="transcript__sticky">
        {/* Plan summary - collapsible */}
        {props.plan.length > 0 && (
          <details className="plan-card" open>
            <summary className="plan-card__title">
              {resolveLanguage(props.language) === 'zh' ? '计划' : 'Plan'} ({props.plan.length})
              <span className="plan-card__status" data-status={props.status}>
                {props.status}
              </span>
            </summary>
            <div className="plan-card__body">
              {props.plan.map((p) => (
                <div key={p.id} className={`plan-card__step plan-card__step--${p.status}`}>
                  <span className="plan-card__icon">
                    {p.status === 'completed' ? '✓' : p.status === 'running' ? '◉' : p.status === 'failed' ? '✕' : '○'}
                  </span>
                  <span className="plan-card__desc">{p.description}</span>
                </div>
              ))}
            </div>
          </details>
        )}

        {/* Logs - collapsible */}
        {props.logs.length > 0 && (
          <details className="log-card">
            <summary className="log-card__title">
              {resolveLanguage(props.language) === 'zh' ? '日志' : 'Logs'} ({props.logs.length})
            </summary>
            <div className="log-card__body">
              {props.logs.slice(-12).map((l) => (
                <div key={l.id} className="log-card__entry">
                  <span className="log-card__time">{new Date(l.time).toLocaleTimeString()}</span>
                  <span className={`log-card__phase log-card__phase--${l.phase}`}>{l.phase}</span>
                  <span className="log-card__event">{l.event}</span>
                  {l.detail && <span className="log-card__detail">{l.detail}</span>}
                </div>
              ))}
            </div>
          </details>
        )}
      </div>

      {/* Scrollable messages */}
      <div className="transcript" ref={ref}>
        {renderMessages(props.messages, props.language)}
      </div>
    </div>
  )
}

// 将连续的 tool 消息分组折叠
function renderMessages(messages: Message[], language: Language) {
  const elements: JSX.Element[] = []
  let i = 0

  while (i < messages.length) {
    const m = messages[i]

    // 收集连续的 tool 消息
    if (m.role === 'tool') {
      const toolGroup: Message[] = []
      while (i < messages.length && messages[i].role === 'tool') {
        toolGroup.push(messages[i])
        i++
      }
      elements.push(
        <details key={`tool-group-${m.id}`} className="msg msg--tool-group" open>
          <summary className="msg__tool-group-summary">
            🔧 {resolveLanguage(language) === 'zh' ? '工具调用' : 'Tool Calls'} ({toolGroup.length})
          </summary>
          <div className="msg__tool-group-body">
            {toolGroup.map((tm) => (
              <div key={tm.id} className="msg msg--tool">
                <div className="msg__tool-header">
                  <span className="msg__icon">🔧</span>
                  <strong>{tm.toolName}</strong>
                </div>
                <pre className="msg__tool-content">{tm.content}</pre>
              </div>
            ))}
          </div>
        </details>
      )
    } else {
      elements.push(<MessageRow key={m.id} message={m} language={language} />)
      i++
    }
  }

  return elements
}

function MessageRow(props: { message: Message; language: Language }) {
  const m = props.message
  if (m.role === 'user') {
    return (
      <div className="msg msg--user">
        <div className="msg__bubble msg__bubble--user">
          <div className="msg__text">{m.content}</div>
          {m.toolCount !== undefined && m.toolCount > 0 && (
            <div className="msg__meta">{m.toolCount} {resolveLanguage(props.language) === 'zh' ? '个工具' : 'tools'}</div>
          )}
        </div>
        <div className="msg__avatar msg__avatar--user">U</div>
      </div>
    )
  }
  if (m.role === 'tool') {
    return (
      <details className="msg msg--tool" open>
        <summary className="msg__tool-summary">
          <span className="msg__icon">🔧</span>
          <strong>{m.toolName}</strong>
        </summary>
        <div className="msg__tool-body">
          <pre className="msg__tool-content">{m.content}</pre>
        </div>
      </details>
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
