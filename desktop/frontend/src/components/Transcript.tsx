import { useEffect, useRef } from 'react'
import type { Message, Language } from '../types'
import { useT } from '../i18n'

interface TranscriptProps {
  language: Language
  messages: Message[]
}

export function Transcript(props: TranscriptProps) {
  const t = useT(props.language)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) {
      ref.current.scrollTop = ref.current.scrollHeight
    }
  }, [props.messages])

  if (props.messages.length === 0) {
    return (
      <div className="transcript__empty">
        <div className="transcript__empty-icon">Z</div>
        <h2 className="transcript__empty-title">{t.emptyTitle}</h2>
        <p className="transcript__empty-desc">{t.emptyDesc}</p>
      </div>
    )
  }

  return (
    <div className="transcript" ref={ref}>
      {props.messages.map((m) => (
        <MessageRow key={m.id} message={m} />
      ))}
    </div>
  )
}

function MessageRow(props: { message: Message }) {
  const m = props.message
  if (m.role === 'system') {
    return (
      <div className="msg msg--system">
        <div className="msg__bubble msg__bubble--system">
          <span className="msg__icon">ℹ</span>
          {m.content}
          <button className="msg__system-action">立即更新</button>
          <button className="msg__system-action msg__system-action--ghost">稍后</button>
        </div>
      </div>
    )
  }
  if (m.role === 'user') {
    return (
      <div className="msg msg--user">
        <div className="msg__bubble msg__bubble--user">
          <div className="msg__text">{m.content}</div>
          {m.toolCount !== undefined && (
            <div className="msg__meta">{m.toolCount} 个工具</div>
          )}
        </div>
        <div className="msg__avatar msg__avatar--user">U</div>
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
